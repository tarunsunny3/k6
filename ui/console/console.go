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
	outMx          *sync.Mutex
	Stdout, Stderr OSFile
	Stdin          io.Reader
	quiet          bool
	theme          *theme
	signalNotify   func(chan<- os.Signal, ...os.Signal)
	signalStop     func(chan<- os.Signal)
	logger         *logrus.Logger
}

func New(
	quiet, colorize bool, termType string,
	signalNotify func(chan<- os.Signal, ...os.Signal),
	signalStop func(chan<- os.Signal),
) *Console {
	outMx := &sync.Mutex{}
	stdout := newConsoleWriter(os.Stdout, outMx, termType)
	stderr := newConsoleWriter(os.Stderr, outMx, termType)
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
		IsTTY:        isTTY,
		outMx:        outMx,
		Stdout:       stdout,
		Stderr:       stderr,
		Stdin:        os.Stdin,
		theme:        th,
		signalNotify: signalNotify,
		signalStop:   signalStop,
		logger:       logger,
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
	OSFile
	isTTY bool
	mutex *sync.Mutex

	// Used for flicker-free persistent objects like the progressbars
	persistentText func()
}

// OSFile is a subset of the functionality implemented by os.File.
type OSFile interface {
	io.Writer
	Fd() uintptr
}

func newConsoleWriter(out OSFile, mx *sync.Mutex, termType string) *consoleWriter {
	isTTY := termType == "dumb" && (isatty.IsTerminal(out.Fd()) || isatty.IsCygwinTerminal(out.Fd()))
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
	n, err = w.OSFile.Write(p)
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
	if c.colorized() {
		return c.theme.foreground.Sprint(s)
	}

	return s
}

func (c *Console) Printf(s string, a ...interface{}) {
	if _, err := fmt.Fprintf(c.Stdout, s, a...); err != nil {
		c.logger.Errorf("could not print '%s' to stdout: %s", s, err.Error())
	}
}

func (c *Console) PrintBanner() {
	_, err := fmt.Fprintf(c.Stdout, "\n%s\n\n", c.ApplyTheme(banner))
	if err != nil {
		c.logger.Warnf("could not print k6 banner message to stdout: %s", err.Error())
	}
}

func (c *Console) PrintYAML(v interface{}) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("could not marshal YAML: %w", err)
	}
	c.Printf(string(data))
	return nil
}

func (c *Console) TermWidth() (int, error) {
	termWidth := defaultTermWidth
	if c.IsTTY {
		tw, _, err := term.GetSize(int(os.Stdout.Fd()))
		if !(tw > 0) || err != nil {
			return termWidth, err
		}
		termWidth = tw
	}

	return termWidth, nil
}

func (c *Console) colorized() bool {
	return c.theme != nil
}

func (c *Console) setPersistentText(pt func()) {
	c.outMx.Lock()
	defer c.outMx.Unlock()

	out := []OSFile{c.Stdout, c.Stderr}
	for _, o := range out {
		cw, ok := o.(*consoleWriter)
		if !ok {
			panic(fmt.Sprintf("expected *consoleWriter; got %T", c.Stdout))
		}
		cw.persistentText = pt
	}
}
