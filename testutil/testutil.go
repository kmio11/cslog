package testutil

import (
	"log/slog"
	"testing"

	"github.com/kmio11/cslog"
)

func SetLogLevel(t *testing.T, level slog.Level) {
	t.Helper()

	logLevelBk := cslog.LogLevel().Level()
	t.Cleanup(func() {
		cslog.SetLogLevel(logLevelBk)
	})

	cslog.SetLogLevel(slog.LevelDebug)
}
