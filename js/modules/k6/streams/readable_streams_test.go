package streams

import (
	"testing"

	"github.com/dop251/goja"
	"go.k6.io/k6/js/compiler"
	"go.k6.io/k6/js/modulestest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstructor(t *testing.T) {
	t.Parallel()

	ts := newConfiguredRuntime(t)

	gotErr := ts.EventLoop.Start(func() error {
		return executeTestScripts(ts.VU.Runtime(), "./tests/readable-streams", "constructor.js")
	})

	assert.NoError(t, gotErr)
}

func TestBadStrategies(t *testing.T) {
	t.Parallel()

	ts := newConfiguredRuntime(t)

	gotErr := ts.EventLoop.Start(func() error {
		return executeTestScripts(ts.VU.Runtime(), "./tests/readable-streams", "bad-strategies.js")
	})

	assert.NoError(t, gotErr)
}

func TestBadUnderlyingSources(t *testing.T) {
	t.Parallel()

	ts := newConfiguredRuntime(t)

	gotErr := ts.EventLoop.Start(func() error {
		return executeTestScripts(ts.VU.Runtime(), "./tests/readable-streams", "bad-underlying-sources.js")
	})

	assert.NoError(t, gotErr)
}

func TestCountQueuingStrategyIntegration(t *testing.T) {
	t.Parallel()

	ts := newConfiguredRuntime(t)

	gotErr := ts.EventLoop.Start(func() error {
		return executeTestScripts(ts.VU.Runtime(), "./tests/readable-streams", "count-queuing-strategy-integration.js")
	})

	assert.NoError(t, gotErr)
}

func TestFloatingPointTotalQueueSize(t *testing.T) {
	t.Parallel()

	ts := newConfiguredRuntime(t)

	gotErr := ts.EventLoop.Start(func() error {
		return executeTestScripts(ts.VU.Runtime(), "./tests/readable-streams", "floating-point-total-queue-size.js")
	})

	assert.NoError(t, gotErr)
}

func TestGeneral(t *testing.T) {
	t.Parallel()

	ts := newConfiguredRuntime(t)

	gotErr := ts.EventLoop.Start(func() error {
		return executeTestScripts(ts.VU.Runtime(), "./tests/readable-streams", "general.js")
	})

	assert.NoError(t, gotErr)
}

func TestPatchedGlobal(t *testing.T) {
	t.Parallel()

	ts := newConfiguredRuntime(t)

	gotErr := ts.EventLoop.Start(func() error {
		return executeTestScripts(ts.VU.Runtime(), "./tests/readable-streams", "patched-global.js")
	})

	assert.NoError(t, gotErr)
}

func TestReentrantStrategies(t *testing.T) {
	t.Parallel()

	ts := newConfiguredRuntime(t)

	gotErr := ts.EventLoop.Start(func() error {
		return executeTestScripts(ts.VU.Runtime(), "./tests/readable-streams", "reentrant-strategies.js")
	})

	assert.NoError(t, gotErr)
}

func newConfiguredRuntime(t testing.TB) *modulestest.Runtime {
	var err error
	runtime := modulestest.NewRuntimeForWPT(t)

	err = runtime.SetupModuleSystem(
		map[string]interface{}{"k6/x/streams": New()},
		nil,
		compiler.New(runtime.VU.InitEnv().Logger),
	)
	require.NoError(t, err)

	m := new(RootModule).NewModuleInstance(runtime.VU)

	// TODO: Can we do this more generic, perhaps even part of the NewRuntimeForWPT signature?
	err = runtime.VU.Runtime().Set("ReadableStream", m.Exports().Named["ReadableStream"])
	err = runtime.VU.Runtime().Set("CountQueuingStrategy", m.Exports().Named["CountQueuingStrategy"])
	require.NoError(t, err)

	return runtime
}

func executeTestScripts(rt *goja.Runtime, base string, scripts ...string) error {
	for _, script := range scripts {
		program, err := modulestest.CompileFile(base, script)
		if err != nil {
			return err
		}

		if _, err = rt.RunProgram(program); err != nil {
			return err
		}
	}

	return nil
}
