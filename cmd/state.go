package cmd

import (
	"context"
	"io"
	"os"
	"os/signal"
	"sync"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

// GlobalState contains the GlobalFlags and accessors for most of the global
// process-external state like CLI arguments, env vars, standard input, output
// and error, etc. In practice, most of it is normally accessed through the `os`
// package from the Go stdlib.
//
// We group them here so we can prevent direct access to them from the rest of
// the k6 codebase. This gives us the ability to mock them and have robust and
// easy-to-write integration-like tests to check the k6 end-to-end behavior in
// any simulated conditions.
//
// `newGlobalState()` returns a globalState object with the real `os`
// parameters, while `newGlobalTestState()` can be used in tests to create
// simulated environments.
type GlobalState struct {
	ctx context.Context

	fs      afero.Fs
	getwd   func() (string, error)
	args    []string
	envVars map[string]string

	defaultFlags, flags globalFlags

	outMutex       *sync.Mutex
	stdOut, stdErr *consoleWriter
	stdIn          io.Reader

	osExit       func(int)
	signalNotify func(chan<- os.Signal, ...os.Signal)
	signalStop   func(chan<- os.Signal)

	logger         *logrus.Logger
	fallbackLogger logrus.FieldLogger
}

// NewGlobalState returns a new GlobalState with the given ctx.
// Ideally, this should be the only function in the whole codebase where we use
// global variables and functions from the os package. Anywhere else, things
// like os.Stdout, os.Stderr, os.Stdin, os.Getenv(), etc. should be removed and
// the respective properties of globalState used instead.
func NewGlobalState(ctx context.Context) *GlobalState {
	isDumbTerm := os.Getenv("TERM") == "dumb"
	stdoutTTY := !isDumbTerm && (isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()))
	stderrTTY := !isDumbTerm && (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()))
	outMutex := &sync.Mutex{}
	stdOut := &consoleWriter{os.Stdout, colorable.NewColorable(os.Stdout), stdoutTTY, outMutex, nil}
	stdErr := &consoleWriter{os.Stderr, colorable.NewColorable(os.Stderr), stderrTTY, outMutex, nil}

	envVars := buildEnvMap(os.Environ())
	_, noColorsSet := envVars["NO_COLOR"] // even empty values disable colors
	logger := &logrus.Logger{
		Out: stdErr,
		Formatter: &logrus.TextFormatter{
			ForceColors:   stderrTTY,
			DisableColors: !stderrTTY || noColorsSet || envVars["K6_NO_COLOR"] != "",
		},
		Hooks: make(logrus.LevelHooks),
		Level: logrus.InfoLevel,
	}

	confDir, err := os.UserConfigDir()
	if err != nil {
		logger.WithError(err).Warn("could not get config directory")
		confDir = ".config"
	}

	defaultFlags := getDefaultFlags(confDir)

	return &GlobalState{
		ctx:          ctx,
		fs:           afero.NewOsFs(),
		getwd:        os.Getwd,
		args:         append(make([]string, 0, len(os.Args)), os.Args...), // copy
		envVars:      envVars,
		defaultFlags: defaultFlags,
		flags:        getFlags(defaultFlags, envVars),
		outMutex:     outMutex,
		stdOut:       stdOut,
		stdErr:       stdErr,
		stdIn:        os.Stdin,
		osExit:       os.Exit,
		signalNotify: signal.Notify,
		signalStop:   signal.Stop,
		logger:       logger,
		fallbackLogger: &logrus.Logger{ // we may modify the other one
			Out:       stdErr,
			Formatter: new(logrus.TextFormatter), // no fancy formatting here
			Hooks:     make(logrus.LevelHooks),
			Level:     logrus.InfoLevel,
		},
	}
}
