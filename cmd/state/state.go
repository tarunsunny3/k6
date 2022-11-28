package state

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"go.k6.io/k6/ui/console"
)

const defaultConfigFileName = "config.json"

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
	ctx                 context.Context
	fs                  afero.Fs
	getWD               func() (string, error)
	cmdArgs             []string
	env                 map[string]string
	defaultFlags, flags globalFlags
	console             *console.Console
	osExit              func(int)
	signalNotify        func(chan<- os.Signal, ...os.Signal)
	signalStop          func(chan<- os.Signal)
	logger              *logrus.Logger
}

// NewGlobalState returns a new GlobalState with the given ctx.
// Ideally, this should be the only function in the whole codebase where we use
// global variables and functions from the os package. Anywhere else, things
// like os.Stdout, os.Stderr, os.Stdin, os.Getenv(), etc. should be removed and
// the respective properties of globalState used instead.
func NewGlobalState(ctx context.Context, cmdArgs []string, env map[string]string) *GlobalState {
	var logger *logrus.Logger
	confDir, err := os.UserConfigDir()
	if err != nil {
		// The logger is initialized in the Console constructor, which so defer
		// logging of this error.
		defer func() {
			logger.WithError(err).Warn("could not get config directory")
		}()
		confDir = ".config"
	}
	defaultFlags := getDefaultFlags(confDir)
	flags := getFlags(defaultFlags, env)

	_, noColorsSet := env["NO_COLOR"] // even empty values disable colors
	colorizeOutput := !noColorsSet || env["K6_NO_COLOR"] == ""
	cons := console.New(flags.quiet, colorizeOutput)
	logger = cons.Logger()

	return &GlobalState{
		ctx:          ctx,
		fs:           afero.NewOsFs(),
		getWD:        os.Getwd,
		cmdArgs:      cmdArgs,
		env:          env,
		defaultFlags: defaultFlags,
		flags:        flags,
		console:      cons,
		osExit:       os.Exit,
		signalNotify: signal.Notify,
		signalStop:   signal.Stop,
		logger:       logger,
	}
}

// globalFlags contains global config values that apply for all k6 sub-commands.
type globalFlags struct {
	configFilePath string
	quiet          bool
	noColor        bool
	address        string
	logOutput      string
	logFormat      string
	verbose        bool
}

func getDefaultFlags(homeDir string) globalFlags {
	return globalFlags{
		address:        "localhost:6565",
		configFilePath: filepath.Join(homeDir, "loadimpact", "k6", defaultConfigFileName),
		logOutput:      "stderr",
	}
}

func getFlags(defaultFlags globalFlags, env map[string]string) globalFlags {
	result := defaultFlags

	// TODO: add env vars for the rest of the values (after adjusting
	// rootCmdPersistentFlagSet(), of course)

	if val, ok := env["K6_CONFIG"]; ok {
		result.configFilePath = val
	}
	if val, ok := env["K6_LOG_OUTPUT"]; ok {
		result.logOutput = val
	}
	if val, ok := env["K6_LOG_FORMAT"]; ok {
		result.logFormat = val
	}
	if env["K6_NO_COLOR"] != "" {
		result.noColor = true
	}
	// Support https://no-color.org/, even an empty value should disable the
	// color output from k6.
	if _, ok := env["NO_COLOR"]; ok {
		result.noColor = true
	}
	return result
}
