package progressbar

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
	"go.k6.io/k6/ui/console"
	"golang.org/x/term"
)

// TODO: show other information here?
// TODO: add a no-progress option that will disable these
// TODO: don't use global variables...
//nolint:funlen,gocognit
func ShowProgress(ctx context.Context, cons *console.Console, pbs []*ProgressBar, logger *logrus.Logger) {
	// Get the longest left side string length, to align progress bars
	// horizontally and trim excess text.
	var leftLen int64
	for _, pb := range pbs {
		l := pb.Left()
		leftLen = max(int64(len(l)), leftLen)
	}
	// Limit to maximum left text length
	maxLeft := int(min(leftLen, maxLeftLength))

	var progressBarsLastRenderLock sync.Mutex
	var progressBarsLastRender []byte

	printProgressBars := func() {
		progressBarsLastRenderLock.Lock()
		_, _ = cons.Stdout.Write(progressBarsLastRender)
		progressBarsLastRenderLock.Unlock()
	}

	var (
		termWidth  = cons.TermWidth()
		widthDelta int
	)
	// Default to responsive progress bars when in an interactive terminal
	renderProgressBars := func(goBack bool) {
		barText, longestLine := renderMultipleBars(
			gs.flags.noColor, gs.stdOut.isTTY, goBack, maxLeft, termWidth, widthDelta, pbs,
		)
		widthDelta = termWidth - longestLine - termPadding
		progressBarsLastRenderLock.Lock()
		progressBarsLastRender = []byte(barText)
		progressBarsLastRenderLock.Unlock()
	}

	// Otherwise fallback to fixed compact progress bars
	if !cons.IsTTY {
		widthDelta = -defaultWidth
		renderProgressBars = func(goBack bool) {
			barText, _ := renderMultipleBars(gs.flags.noColor, gs.stdOut.isTTY, goBack, maxLeft, termWidth, widthDelta, pbs)
			progressBarsLastRenderLock.Lock()
			progressBarsLastRender = []byte(barText)
			progressBarsLastRenderLock.Unlock()
		}
	}

	// TODO: make configurable?
	updateFreq := 1 * time.Second
	var stdoutFD int
	if cons.IsTTY {
		stdoutFD = int(cons.Stdout.Fd())
		updateFreq = 100 * time.Millisecond
		gs.ui.setPersistentText(printProgressBars)
		defer gs.ui.setPersistentText(nil)
		// gs.outMutex.Lock()
		// gs.stdOut.persistentText = printProgressBars
		// gs.stdErr.persistentText = printProgressBars
		// gs.outMutex.Unlock()
		// defer func() {
		// 	gs.outMutex.Lock()
		// 	gs.stdOut.persistentText = nil
		// 	gs.stdErr.persistentText = nil
		// 	gs.outMutex.Unlock()
		// }()
	}

	var winch chan os.Signal
	if sig := getWinchSignal(); sig != nil {
		winch = make(chan os.Signal, 10)
		gs.signalNotify(winch, sig)
		defer gs.signalStop(winch)
	}

	ticker := time.NewTicker(updateFreq)
	ctxDone := ctx.Done()
	for {
		select {
		case <-ctxDone:
			renderProgressBars(false)
			gs.outMutex.Lock()
			printProgressBars()
			gs.outMutex.Unlock()
			return
		case <-winch:
			if gs.stdOut.isTTY && !errTermGetSize {
				// More responsive progress bar resizing on platforms with SIGWINCH (*nix)
				tw, _, err := term.GetSize(stdoutFD)
				if tw > 0 && err == nil {
					termWidth = tw
				}
			}
		case <-ticker.C:
			// Default ticker-based progress bar resizing
			if gs.stdOut.isTTY && !errTermGetSize && winch == nil {
				tw, _, err := term.GetSize(stdoutFD)
				if tw > 0 && err == nil {
					termWidth = tw
				}
			}
		}
		renderProgressBars(true)
		gs.ui.Print()
		gs.outMutex.Lock()
		printProgressBars()
		gs.outMutex.Unlock()
	}
}

//nolint:funlen
func renderMultipleBars(
	nocolor, isTTY, goBack bool, maxLeft, termWidth, widthDelta int, pbs []*pb.ProgressBar,
) (string, int) {
	lineEnd := "\n"
	if isTTY {
		// TODO: check for cross platform support
		lineEnd = "\x1b[K\n" // erase till end of line
	}

	var (
		// Amount of times line lengths exceed termWidth.
		// Needed to factor into the amount of lines to jump
		// back with [A and avoid scrollback issues.
		lineBreaks  int
		longestLine int
		// Maximum length of each right side column except last,
		// used to calculate the padding between columns.
		maxRColumnLen = make([]int, 2)
		pbsCount      = len(pbs)
		rendered      = make([]pb.ProgressBarRender, pbsCount)
		result        = make([]string, pbsCount+2)
	)

	result[0] = lineEnd // start with an empty line

	// First pass to render all progressbars and get the maximum
	// lengths of right-side columns.
	for i, pb := range pbs {
		rend := pb.Render(maxLeft, widthDelta)
		for i := range rend.Right {
			// Skip last column, since there's nothing to align after it (yet?).
			if i == len(rend.Right)-1 {
				break
			}
			if len(rend.Right[i]) > maxRColumnLen[i] {
				maxRColumnLen[i] = len(rend.Right[i])
			}
		}
		rendered[i] = rend
	}

	// Second pass to render final output, applying padding where needed
	for i := range rendered {
		rend := rendered[i]
		if rend.Hijack != "" {
			result[i+1] = rend.Hijack + lineEnd
			runeCount := utf8.RuneCountInString(rend.Hijack)
			lineBreaks += (runeCount - termPadding) / termWidth
			continue
		}
		var leftText, rightText string
		leftPadFmt := fmt.Sprintf("%%-%ds", maxLeft)
		leftText = fmt.Sprintf(leftPadFmt, rend.Left)
		for i := range rend.Right {
			rpad := 0
			if len(maxRColumnLen) > i {
				rpad = maxRColumnLen[i]
			}
			rightPadFmt := fmt.Sprintf(" %%-%ds", rpad+1)
			rightText += fmt.Sprintf(rightPadFmt, rend.Right[i])
		}
		// Get visible line length, without ANSI escape sequences (color)
		status := fmt.Sprintf(" %s ", rend.Status())
		line := leftText + status + rend.Progress() + rightText
		lineRuneCount := utf8.RuneCountInString(line)
		if lineRuneCount > longestLine {
			longestLine = lineRuneCount
		}
		lineBreaks += (lineRuneCount - termPadding) / termWidth
		if !nocolor {
			rend.Color = true
			status = fmt.Sprintf(" %s ", rend.Status())
			line = fmt.Sprintf(leftPadFmt+"%s%s%s",
				rend.Left, status, rend.Progress(), rightText)
		}
		result[i+1] = line + lineEnd
	}

	if isTTY && goBack {
		// Clear screen and go back to the beginning
		// TODO: check for cross platform support
		result[pbsCount+1] = fmt.Sprintf("\r\x1b[J\x1b[%dA", pbsCount+lineBreaks+1)
	} else {
		result[pbsCount+1] = ""
	}

	return strings.Join(result, ""), longestLine
}
