package crawler

import (
	"fmt"
	"io"
	"os"
)

// To capture debug output in tests, construct the runner and set its embedded
// debugLogger fields directly (same package access):
//
//	r.debugLogger = debugLogger{debug: true, debugTo: &myBuf}

// debugLogger can be embedded in a runner to provide uniform debug logging.
// Runners that embed it automatically satisfy Runner.SetDebug.
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

func (d *debugLogger) debugf(format string, args ...any) {
	if !d.debug {
		return
	}
	_, _ = fmt.Fprintf(d.debugTo, "[debug] "+format+"\n", args...)
}
