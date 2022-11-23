package cmd

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.k6.io/k6/lib/testutils"
)

func TestDeprecatedOptionWarning(t *testing.T) {
	t.Parallel()

	ts := newGlobalTestState(t)
	ts.args = []string{"k6", "--logformat", "json", "run", "-"}
	ts.stdIn = bytes.NewBuffer([]byte(`
		console.log('foo');
		export default function() { console.log('bar'); };
	`))

	newRootCommand(ts.globalState).execute()

	logMsgs := ts.loggerHook.Drain()
	assert.True(t, testutils.LogContains(logMsgs, logrus.InfoLevel, "foo"))
	assert.True(t, testutils.LogContains(logMsgs, logrus.InfoLevel, "bar"))
	assert.Contains(t, ts.stdErr.String(), `"level":"info","msg":"foo","source":"console"`)
	assert.Contains(t, ts.stdErr.String(), `"level":"info","msg":"bar","source":"console"`)

	// TODO: after we get rid of cobra, actually emit this message to stderr
	// and, ideally, through the log, not just print it...
	assert.False(t, testutils.LogContains(logMsgs, logrus.InfoLevel, "logformat"))
	assert.Contains(t, ts.stdOut.String(), `--logformat has been deprecated`)
}
