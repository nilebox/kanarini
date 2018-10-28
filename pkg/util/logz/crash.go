package logz

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"time"
)

func LogStructuredPanic() {
	if r := recover(); r != nil {
		logStructuredPanic(os.Stderr, r, time.Now(), debug.Stack())
		panic(r)
	}
}

// internally zap is overly opinionated about what to do when the log level is fatal or panic
// it chooses to call os.exit or panic if the level is set. There does not appear to be a simple
// way to work around that choice so we build the log message by hand instead.
func logStructuredPanic(out io.Writer, panicValue interface{}, now time.Time, stack []byte) {
	bytes, err := json.Marshal(struct {
		Level   string `json:"level"`
		Time    string `json:"time"`
		Message string `json:"msg"`
		Stack   string `json:"stack"`
	}{
		Level:   "fatal",
		Time:    now.Format(time.RFC3339),
		Message: fmt.Sprintf("%v", panicValue),
		Stack:   string(stack),
	})
	if err != nil {
		fmt.Fprintf(out, "error while serializing panic: %+v\n", err) // nolint: errcheck, gas
		fmt.Fprintf(out, "original panic: %+v\n", panicValue)         // nolint: errcheck, gas
		return
	}
	fmt.Fprintf(out, "%s\n", bytes) // nolint: errcheck, gas
}
