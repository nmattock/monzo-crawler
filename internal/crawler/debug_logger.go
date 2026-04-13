package crawler

import (
	"fmt"
	"io"
	"os"
)

// debugLogger can be embedded in a runner to provide uniform debug logging.
// Runners that embed it automatically satisfy DebuggableRunner.SetDebug.
type debugLogger struct {
	debug   bool
	debugTo io.Writer
}

func newDebugLogger() debugLogger {
	return debugLogger{debugTo: os.Stderr}
}

func (d *debugLogger) SetDebug(enabled bool) {
	d.debug = enabled
}

// SetDebugOutput sets where debug logs are written. Defaults to stderr.
func (d *debugLogger) SetDebugOutput(w io.Writer) {
	if w == nil {
		d.debugTo = os.Stderr
		return
	}
	d.debugTo = w
}

func (d *debugLogger) debugf(format string, args ...any) {
	if !d.debug {
		return
	}
	_, _ = fmt.Fprintf(d.debugTo, "[debug] "+format+"\n", args...)
}
