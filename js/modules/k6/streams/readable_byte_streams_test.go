package streams

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// FIXME: Re-name, or use table-driven.
func TestGeneral2(t *testing.T) {
	t.Parallel()

	ts := newConfiguredRuntime(t)

	gotErr := ts.EventLoop.Start(func() error {
		return executeTestScripts(ts.VU.Runtime(), "./tests/readable-byte-streams", "general.js")
	})

	assert.NoError(t, gotErr)
}
