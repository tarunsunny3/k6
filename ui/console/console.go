package console

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/sirupsen/logrus"
	"golang.org/x/term"

	"gopkg.in/yaml.v3"
)

// Console enables synced writing to stdout and stderr ...
type Console struct {
	IsTTY          bool
	writeMx        *sync.Mutex
	Stdout, Stderr OSFile
	Stdin          io.Reader
	quiet          bool
	theme          *theme
	logger         *logrus.Logger
}

func New(quiet, colorize bool) *Console {
	writeMx := &sync.Mutex{}
	stdout := newConsoleWriter(os.Stdout, writeMx)
	stderr := newConsoleWriter(os.Stderr, writeMx)
	isTTY := stdout.isTTY && stderr.isTTY

	// Default logger without any formatting
	logger := &logrus.Logger{
		Out:       stderr,
		Formatter: new(logrus.TextFormatter), // no fancy formatting here
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.InfoLevel,
	}

	var th *theme
	// Only enable themes and a fancy logger if we're in a TTY
	if isTTY && colorize {
		th = &theme{foreground: getColor(color.FgCyan)}

		logger = &logrus.Logger{
			Out: stderr,
			Formatter: &logrus.TextFormatter{
				ForceColors:   true,
				DisableColors: false,
			},
			Hooks: make(logrus.LevelHooks),
			Level: logrus.InfoLevel,
		}
	}

	return &Console{
		IsTTY:   isTTY,
		writeMx: writeMx,
		Stdout:  stdout,
		Stderr:  stderr,
		Stdin:   os.Stdin,
		theme:   th,
		logger:  logger,
	}
}

func (c *Console) Logger() *logrus.Logger {
	return c.logger
}

type theme struct {
	foreground *color.Color
}

// A writer that syncs writes with a mutex and, if the output is a TTY, clears before newlines.
type consoleWriter struct {
	io.Writer
	isTTY bool
	mutex *sync.Mutex

	// Used for flicker-free persistent objects like the progressbars
	persistentText func()
}

type OSFile interface {
	io.Writer
	Fd() uintptr
}

func newConsoleWriter(out OSFile, mx *sync.Mutex) *consoleWriter {
	isDumbTerm := os.Getenv("TERM") == "dumb"
	isTTY := !isDumbTerm && (isatty.IsTerminal(out.Fd()) || isatty.IsCygwinTerminal(out.Fd()))
	return &consoleWriter{out, isTTY, mx, nil}
}

func (w *consoleWriter) Write(p []byte) (n int, err error) {
	origLen := len(p)
	if w.isTTY {
		// Add a TTY code to erase till the end of line with each new line
		// TODO: check how cross-platform this is...
		p = bytes.ReplaceAll(p, []byte{'\n'}, []byte{'\x1b', '[', '0', 'K', '\n'})
	}

	w.mutex.Lock()
	n, err = w.Writer.Write(p)
	if w.persistentText != nil {
		w.persistentText()
	}
	w.mutex.Unlock()

	if err != nil && n < origLen {
		return n, err
	}
	return origLen, err
}

// getColor returns the requested color, or an uncolored object, depending on
// the value of noColor. The explicit EnableColor() and DisableColor() are
// needed because the library checks os.Stdout itself otherwise...
func getColor(attributes ...color.Attribute) *color.Color {
	// if noColor {
	// 	c := color.New()
	// 	c.DisableColor()
	// 	return c
	// }

	c := color.New(attributes...)
	c.EnableColor()
	return c
}

func (c *Console) ApplyTheme(s string) string {
	if c.theme != nil {
		return c.theme.foreground.Sprint(s)
	}

	return s
}

func (c *Console) GetWinchSignal(s string) os.Signal {
	return getWinchSignal()
}

func (c *Console) Print(s string) {
	if _, err := fmt.Fprint(c.Stdout, s); err != nil {
		c.logger.Errorf("could not print '%s' to stdout: %s", s, err.Error())
	}
}

func (c *Console) PrintBanner() {
	_, err := fmt.Fprintf(c.Stdout, "\n%s\n\n", c.ApplyTheme(banner))
	if err != nil {
		c.logger.Warnf("could not print k6 banner message to stdout: %s", err.Error())
	}
}

func (c *Console) TermWidth() int {
	termWidth := DefaultTermWidth
	if c.IsTTY {
		tw, _, err := term.GetSize(int(os.Stdout.Fd()))
		if !(tw > 0) || err != nil {
			c.logger.WithError(err).Warn("error getting terminal size")
		} else {
			termWidth = tw
		}
	}

	return termWidth
}

func yamlPrint(w io.Writer, v interface{}) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("could not marshal YAML: %w", err)
	}
	_, err = fmt.Fprint(w, string(data))
	if err != nil {
		return fmt.Errorf("could flush the data to the output: %w", err)
	}
	return nil
}
