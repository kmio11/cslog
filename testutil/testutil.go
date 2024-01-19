package testutil

import (
	"fmt"
	"log/slog"
	"testing"

	"github.com/kmio11/cslog"
)

// RemoveTime removes the top-level time attribute.
// It is intended to be used as a ReplaceAttr function,
// to make example output deterministic.
func RemoveTime(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		return slog.Attr{}
	}
	return a
}

func SetLogLevel(t *testing.T, level slog.Level) {
	t.Helper()

	logLevelBk := cslog.LogLevel().Level()
	t.Cleanup(func() {
		cslog.SetLogLevel(logLevelBk)
	})

	cslog.SetLogLevel(slog.LevelDebug)
}

var _ (cslog.IDGenerator) = (*CountUpIDGen)(nil)

// CountUpIDGen is IDGenerator to output fixed value.
type CountUpIDGen struct {
	cnt int
}

func (gen *CountUpIDGen) NewID() cslog.LogID {
	id := fmt.Sprintf("%016d", gen.cnt)
	gen.cnt += 1
	return cslog.StringLogID(id)
}

func SetIDGen(t *testing.T) {
	t.Helper()
	cslog.SetLogIdGenerator(&CountUpIDGen{})
}

func ResetIDCount(t *testing.T) {
	t.Helper()
	cslog.SetLogIdGenerator(&CountUpIDGen{})
}
