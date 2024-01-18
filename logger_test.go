package cslog_test

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/kmio11/cslog"
	"github.com/kmio11/cslog/testutil"
)

// textTimeRE is a regexp to match log timestamps for Text handler.
// This is RFC3339Nano with the fixed 3 digit sub-second precision.
const textTimeRE = `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}(Z|[+-]\d{2}:\d{2})`

func checkLogOutput(t *testing.T, got, wantRegexp string) {
	t.Helper()
	got = clean(got)
	wantRegexp = "^" + wantRegexp + "$"
	matched, err := regexp.MatchString(wantRegexp, got)
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Errorf("\ngot  %s\nwant %s", got, wantRegexp)
	}
}

// clean prepares log output for comparison.
func clean(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	return strings.ReplaceAll(s, "\n", "~")
}

func TestLogger(t *testing.T) {
	buf := testutil.UseBuf(t, false)

	check := func(t *testing.T, want string) {
		t.Helper()
		if want != "" {
			want = "time=" + textTimeRE + " " + want
		}
		checkLogOutput(t, buf.String(), want)
		buf.Reset()
	}

	t.Run("logger_parent", func(t *testing.T) {
		testutil.SetIDGen(t)

		ctx, logger := cslog.GetContextLogger(context.Background())

		// By default, debug messages are not printed.
		logger.Debug("debug", slog.Int("a", 1), "b", 2)
		check(t, "")

		testutil.SetLogLevel(t, slog.LevelDebug)

		logger.Debug("debug", slog.Int("a", 1), "b", 2)
		check(t, "level=DEBUG msg=debug a=1 b=2 logId=3030303030303030")

		logger.Warn("w", slog.Duration("dur", 3*time.Second))
		check(t, `level=WARN msg=w dur=3s logId=3030303030303030`)

		logger.Error("bad", "a", 1)
		check(t, `level=ERROR msg=bad a=1 logId=3030303030303030`)

		logger.Log(ctx, slog.LevelWarn+1, "w", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=WARN\+1 msg=w a=1 b=two logId=3030303030303030`)

		logger.LogAttrs(ctx, slog.LevelInfo+1, "a b c", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=INFO\+1 msg="a b c" a=1 b=two logId=3030303030303030`)

		logger.Info("info", "a", []slog.Attr{slog.Int("i", 1)})
		check(t, `level=INFO msg=info a.i=1 logId=3030303030303030`)

		logger.Info("info", "a", slog.GroupValue(slog.Int("i", 1)))
		check(t, `level=INFO msg=info a.i=1 logId=3030303030303030`)
	})

	t.Run("logger_child", func(t *testing.T) {
		testutil.SetIDGen(t)

		parentCtx, _ := cslog.GetContextLogger(context.Background())
		ctx, childLogger := cslog.CreateChildContextLogger(parentCtx)

		// By default, debug messages are not printed.
		childLogger.Debug("debug", slog.Int("a", 1), "b", 2)
		check(t, "")

		testutil.SetLogLevel(t, slog.LevelDebug)

		childLogger.Debug("debug", slog.Int("a", 1), "b", 2)
		check(t, "level=DEBUG msg=debug a=1 b=2 logId=3030303030303031 parentLogId=3030303030303030")

		childLogger.Warn("w", slog.Duration("dur", 3*time.Second))
		check(t, `level=WARN msg=w dur=3s logId=3030303030303031 parentLogId=3030303030303030`)

		childLogger.Error("bad", "a", 1)
		check(t, `level=ERROR msg=bad a=1 logId=3030303030303031 parentLogId=3030303030303030`)

		childLogger.Log(ctx, slog.LevelWarn+1, "w", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=WARN\+1 msg=w a=1 b=two logId=3030303030303031 parentLogId=3030303030303030`)

		childLogger.LogAttrs(ctx, slog.LevelInfo+1, "a b c", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=INFO\+1 msg="a b c" a=1 b=two logId=3030303030303031 parentLogId=3030303030303030`)

		childLogger.Info("info", "a", []slog.Attr{slog.Int("i", 1)})
		check(t, `level=INFO msg=info a.i=1 logId=3030303030303031 parentLogId=3030303030303030`)

		childLogger.Info("info", "a", slog.GroupValue(slog.Int("i", 1)))
		check(t, `level=INFO msg=info a.i=1 logId=3030303030303031 parentLogId=3030303030303030`)
	})

	t.Run("logger_context", func(t *testing.T) {
		testutil.SetIDGen(t)

		parentCtx, _ := cslog.GetContextLogger(context.Background())
		ctx, childLogger := cslog.CreateChildContextLogger(parentCtx)

		// If ctx has logId/parentLogId, logger's logId/parentLogId is overwritten.
		ctx = cslog.WithChildLogContext(ctx)

		// By default, debug messages are not printed.
		childLogger.DebugContext(ctx, "debug", slog.Int("a", 1), "b", 2)
		check(t, "")

		testutil.SetLogLevel(t, slog.LevelDebug)

		childLogger.DebugContext(ctx, "debug", slog.Int("a", 1), "b", 2)
		check(t, "level=DEBUG msg=debug a=1 b=2 logId=3030303030303032 parentLogId=3030303030303031")

		childLogger.WarnContext(ctx, "w", slog.Duration("dur", 3*time.Second))
		check(t, `level=WARN msg=w dur=3s logId=3030303030303032 parentLogId=3030303030303031`)

		childLogger.ErrorContext(ctx, "bad", "a", 1)
		check(t, `level=ERROR msg=bad a=1 logId=3030303030303032 parentLogId=3030303030303031`)

		childLogger.Log(ctx, slog.LevelWarn+1, "w", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=WARN\+1 msg=w a=1 b=two logId=3030303030303032 parentLogId=3030303030303031`)

		childLogger.LogAttrs(ctx, slog.LevelInfo+1, "a b c", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=INFO\+1 msg="a b c" a=1 b=two logId=3030303030303032 parentLogId=3030303030303031`)

		childLogger.InfoContext(ctx, "info", "a", []slog.Attr{slog.Int("i", 1)})
		check(t, `level=INFO msg=info a.i=1 logId=3030303030303032 parentLogId=3030303030303031`)

		childLogger.InfoContext(ctx, "info", "a", slog.GroupValue(slog.Int("i", 1)))
		check(t, `level=INFO msg=info a.i=1 logId=3030303030303032 parentLogId=3030303030303031`)
	})

	t.Run("context", func(t *testing.T) {
		testutil.SetIDGen(t)

		ctx := cslog.WithLogContext(context.Background())

		// By default, debug messages are not printed.
		cslog.DebugContext(ctx, "debug", slog.Int("a", 1), "b", 2)
		check(t, "")

		testutil.SetLogLevel(t, slog.LevelDebug)

		cslog.DebugContext(ctx, "debug", slog.Int("a", 1), "b", 2)
		check(t, "level=DEBUG msg=debug a=1 b=2 logId=3030303030303030")

		cslog.WarnContext(ctx, "w", slog.Duration("dur", 3*time.Second))
		check(t, `level=WARN msg=w dur=3s logId=3030303030303030`)

		cslog.ErrorContext(ctx, "bad", "a", 1)
		check(t, `level=ERROR msg=bad a=1 logId=3030303030303030`)

		cslog.Log(ctx, slog.LevelWarn+1, "w", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=WARN\+1 msg=w a=1 b=two logId=3030303030303030`)

		cslog.LogAttrs(ctx, slog.LevelInfo+1, "a b c", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=INFO\+1 msg="a b c" a=1 b=two logId=3030303030303030`)

		cslog.InfoContext(ctx, "info", "a", []slog.Attr{slog.Int("i", 1)})
		check(t, `level=INFO msg=info a.i=1 logId=3030303030303030`)

		cslog.InfoContext(ctx, "info", "a", slog.GroupValue(slog.Int("i", 1)))
		check(t, `level=INFO msg=info a.i=1 logId=3030303030303030`)
	})

	t.Run("context_child", func(t *testing.T) {
		testutil.SetIDGen(t)

		ctx := cslog.WithLogContext(context.Background())
		childCtx := cslog.WithChildLogContext(ctx)

		// By default, debug messages are not printed.
		cslog.DebugContext(childCtx, "debug", slog.Int("a", 1), "b", 2)
		check(t, "")

		testutil.SetLogLevel(t, slog.LevelDebug)

		cslog.DebugContext(childCtx, "debug", slog.Int("a", 1), "b", 2)
		check(t, "level=DEBUG msg=debug a=1 b=2 logId=3030303030303031 parentLogId=3030303030303030")

		cslog.WarnContext(childCtx, "w", slog.Duration("dur", 3*time.Second))
		check(t, `level=WARN msg=w dur=3s logId=3030303030303031 parentLogId=3030303030303030`)

		cslog.ErrorContext(childCtx, "bad", "a", 1)
		check(t, `level=ERROR msg=bad a=1 logId=3030303030303031 parentLogId=3030303030303030`)

		cslog.Log(childCtx, slog.LevelWarn+1, "w", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=WARN\+1 msg=w a=1 b=two logId=3030303030303031 parentLogId=3030303030303030`)

		cslog.LogAttrs(childCtx, slog.LevelInfo+1, "a b c", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=INFO\+1 msg="a b c" a=1 b=two logId=3030303030303031 parentLogId=3030303030303030`)

		cslog.InfoContext(childCtx, "info", "a", []slog.Attr{slog.Int("i", 1)})
		check(t, `level=INFO msg=info a.i=1 logId=3030303030303031 parentLogId=3030303030303030`)

		cslog.InfoContext(childCtx, "info", "a", slog.GroupValue(slog.Int("i", 1)))
		check(t, `level=INFO msg=info a.i=1 logId=3030303030303031 parentLogId=3030303030303030`)
	})
}
