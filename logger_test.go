package cslog_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/kmio11/cslog"
	"github.com/kmio11/cslog/testutil"
)

// textTimeRE is a regexp to match log timestamps for Text handler.
// This is RFC3339Nano with the fixed 3 digit sub-second precision.
const textTimeRE = `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}(Z|[+-]\d{2}:\d{2})`

func TestLog(t *testing.T) {

	h := testutil.NewAssertHandler(t, testutil.AssertHandlerOpts{})
	cslog.SetInnerHandler(h)

	check := func(t *testing.T, want string) {
		t.Helper()
		if want != "" {
			want = "time=" + textTimeRE + " " + want
		}
		h.Check(t, want)
	}

	t.Run("context", func(t *testing.T) {
		testutil.SetIDGen(t)

		ctx := cslog.WithLogContext(context.Background())

		// By default, debug messages are not printed.
		cslog.DebugContext(ctx, "debug", slog.Int("a", 1), "b", 2)
		check(t, "")

		testutil.SetLogLevel(t, slog.LevelDebug)

		cslog.DebugContext(ctx, "debug", slog.Int("a", 1), "b", 2)
		check(t, "level=DEBUG msg=debug a=1 b=2 logId=0000000000000000")

		cslog.WarnContext(ctx, "w", slog.Duration("dur", 3*time.Second))
		check(t, `level=WARN msg=w dur=3s logId=0000000000000000`)

		cslog.ErrorContext(ctx, "bad", "a", 1)
		check(t, `level=ERROR msg=bad a=1 logId=0000000000000000`)

		cslog.Log(ctx, slog.LevelWarn+1, "w", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=WARN\+1 msg=w a=1 b=two logId=0000000000000000`)

		cslog.LogAttrs(ctx, slog.LevelInfo+1, "a b c", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=INFO\+1 msg="a b c" a=1 b=two logId=0000000000000000`)

		cslog.InfoContext(ctx, "info", "a", []slog.Attr{slog.Int("i", 1)})
		check(t, `level=INFO msg=info a.i=1 logId=0000000000000000`)

		cslog.InfoContext(ctx, "info", "a", slog.GroupValue(slog.Int("i", 1)))
		check(t, `level=INFO msg=info a.i=1 logId=0000000000000000`)
	})

	t.Run("without_context", func(t *testing.T) {
		testutil.SetIDGen(t)

		_ = cslog.WithLogContext(context.Background())

		// By default, debug messages are not printed.
		cslog.Debug("debug", slog.Int("a", 1), "b", 2)
		check(t, "")

		testutil.SetLogLevel(t, slog.LevelDebug)

		cslog.Debug("debug", slog.Int("a", 1), "b", 2)
		check(t, "level=DEBUG msg=debug a=1 b=2")

		cslog.Warn("w", slog.Duration("dur", 3*time.Second))
		check(t, `level=WARN msg=w dur=3s`)

		cslog.Error("bad", "a", 1)
		check(t, `level=ERROR msg=bad a=1`)

		cslog.Info("info", "a", []slog.Attr{slog.Int("i", 1)})
		check(t, `level=INFO msg=info a.i=1`)

		cslog.Info("info", "a", slog.GroupValue(slog.Int("i", 1)))
		check(t, `level=INFO msg=info a.i=1`)
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
		check(t, "level=DEBUG msg=debug a=1 b=2 logId=0000000000000001 parentLogId=0000000000000000")

		cslog.WarnContext(childCtx, "w", slog.Duration("dur", 3*time.Second))
		check(t, `level=WARN msg=w dur=3s logId=0000000000000001 parentLogId=0000000000000000`)

		cslog.ErrorContext(childCtx, "bad", "a", 1)
		check(t, `level=ERROR msg=bad a=1 logId=0000000000000001 parentLogId=0000000000000000`)

		cslog.Log(childCtx, slog.LevelWarn+1, "w", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=WARN\+1 msg=w a=1 b=two logId=0000000000000001 parentLogId=0000000000000000`)

		cslog.LogAttrs(childCtx, slog.LevelInfo+1, "a b c", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=INFO\+1 msg="a b c" a=1 b=two logId=0000000000000001 parentLogId=0000000000000000`)

		cslog.InfoContext(childCtx, "info", "a", []slog.Attr{slog.Int("i", 1)})
		check(t, `level=INFO msg=info a.i=1 logId=0000000000000001 parentLogId=0000000000000000`)

		cslog.InfoContext(childCtx, "info", "a", slog.GroupValue(slog.Int("i", 1)))
		check(t, `level=INFO msg=info a.i=1 logId=0000000000000001 parentLogId=0000000000000000`)
	})

	t.Run("custom_attr", func(t *testing.T) {
		testutil.SetIDGen(t)

		type ctxKey struct{}
		cslog.AddContextAttrs(
			cslog.Context("ctxAttr", nil, cslog.GetStringFn(ctxKey{})),
		)

		ctx := context.Background()

		cslog.ErrorContext(ctx, "no value in context", "a", 1)
		check(t, `level=ERROR msg="no value in context" a=1`)

		ctx = context.WithValue(ctx, ctxKey{}, "testValue")
		cslog.ErrorContext(ctx, "value exists in context", "a", 1)
		check(t, `level=ERROR msg="value exists in context" a=1 ctxAttr=testValue`)

		ctx = cslog.WithLogContext(ctx)
		cslog.ErrorContext(ctx, "with logId", "a", 1)
		check(t, `level=ERROR msg="with logId" a=1 logId=0000000000000000 ctxAttr=testValue`)

		ctx = cslog.WithChildLogContext(ctx)
		cslog.ErrorContext(ctx, "with logId and parentLogId", "a", 1)
		check(t, `level=ERROR msg="with logId and parentLogId" a=1 logId=0000000000000001 parentLogId=0000000000000000 ctxAttr=testValue`)
	})
}

func TestNewLoggerWithContext(t *testing.T) {
	h := testutil.NewAssertHandler(t, testutil.AssertHandlerOpts{})
	cslog.SetInnerHandler(h)

	check := func(t *testing.T, want string) {
		t.Helper()
		if want != "" {
			want = "time=" + textTimeRE + " " + want
		}
		h.Check(t, want)
	}

	t.Run("logger_parent", func(t *testing.T) {
		testutil.SetIDGen(t)

		ctx, logger := cslog.NewLoggerWithContext(context.Background())

		// By default, debug messages are not printed.
		logger.Debug("debug", slog.Int("a", 1), "b", 2)
		check(t, "")

		testutil.SetLogLevel(t, slog.LevelDebug)

		logger.Debug("debug", slog.Int("a", 1), "b", 2)
		check(t, "level=DEBUG msg=debug a=1 b=2 logId=0000000000000000")

		logger.Warn("w", slog.Duration("dur", 3*time.Second))
		check(t, `level=WARN msg=w dur=3s logId=0000000000000000`)

		logger.Error("bad", "a", 1)
		check(t, `level=ERROR msg=bad a=1 logId=0000000000000000`)

		logger.Log(ctx, slog.LevelWarn+1, "w", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=WARN\+1 msg=w a=1 b=two logId=0000000000000000`)

		logger.LogAttrs(ctx, slog.LevelInfo+1, "a b c", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=INFO\+1 msg="a b c" a=1 b=two logId=0000000000000000`)

		logger.Info("info", "a", []slog.Attr{slog.Int("i", 1)})
		check(t, `level=INFO msg=info a.i=1 logId=0000000000000000`)

		logger.Info("info", "a", slog.GroupValue(slog.Int("i", 1)))
		check(t, `level=INFO msg=info a.i=1 logId=0000000000000000`)
	})

	t.Run("logger_child", func(t *testing.T) {
		testutil.SetIDGen(t)

		parentCtx, _ := cslog.NewLoggerWithContext(context.Background())
		ctx, childLogger := cslog.NewLoggerWithChildContext(parentCtx)

		// By default, debug messages are not printed.
		childLogger.Debug("debug", slog.Int("a", 1), "b", 2)
		check(t, "")

		testutil.SetLogLevel(t, slog.LevelDebug)

		childLogger.Debug("debug", slog.Int("a", 1), "b", 2)
		check(t, "level=DEBUG msg=debug a=1 b=2 logId=0000000000000001 parentLogId=0000000000000000")

		childLogger.Warn("w", slog.Duration("dur", 3*time.Second))
		check(t, `level=WARN msg=w dur=3s logId=0000000000000001 parentLogId=0000000000000000`)

		childLogger.Error("bad", "a", 1)
		check(t, `level=ERROR msg=bad a=1 logId=0000000000000001 parentLogId=0000000000000000`)

		childLogger.Log(ctx, slog.LevelWarn+1, "w", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=WARN\+1 msg=w a=1 b=two logId=0000000000000001 parentLogId=0000000000000000`)

		childLogger.LogAttrs(ctx, slog.LevelInfo+1, "a b c", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=INFO\+1 msg="a b c" a=1 b=two logId=0000000000000001 parentLogId=0000000000000000`)

		childLogger.Info("info", "a", []slog.Attr{slog.Int("i", 1)})
		check(t, `level=INFO msg=info a.i=1 logId=0000000000000001 parentLogId=0000000000000000`)

		childLogger.Info("info", "a", slog.GroupValue(slog.Int("i", 1)))
		check(t, `level=INFO msg=info a.i=1 logId=0000000000000001 parentLogId=0000000000000000`)
	})

	t.Run("logger_context", func(t *testing.T) {
		testutil.SetIDGen(t)

		parentCtx, _ := cslog.NewLoggerWithContext(context.Background())
		ctx, childLogger := cslog.NewLoggerWithChildContext(parentCtx)

		// If ctx has logId/parentLogId, logger's logId/parentLogId is overwritten.
		ctx = cslog.WithChildLogContext(ctx)

		// By default, debug messages are not printed.
		childLogger.DebugContext(ctx, "debug", slog.Int("a", 1), "b", 2)
		check(t, "")

		testutil.SetLogLevel(t, slog.LevelDebug)

		childLogger.DebugContext(ctx, "debug", slog.Int("a", 1), "b", 2)
		check(t, "level=DEBUG msg=debug a=1 b=2 logId=0000000000000002 parentLogId=0000000000000001")

		childLogger.WarnContext(ctx, "w", slog.Duration("dur", 3*time.Second))
		check(t, `level=WARN msg=w dur=3s logId=0000000000000002 parentLogId=0000000000000001`)

		childLogger.ErrorContext(ctx, "bad", "a", 1)
		check(t, `level=ERROR msg=bad a=1 logId=0000000000000002 parentLogId=0000000000000001`)

		childLogger.Log(ctx, slog.LevelWarn+1, "w", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=WARN\+1 msg=w a=1 b=two logId=0000000000000002 parentLogId=0000000000000001`)

		childLogger.LogAttrs(ctx, slog.LevelInfo+1, "a b c", slog.Int("a", 1), slog.String("b", "two"))
		check(t, `level=INFO\+1 msg="a b c" a=1 b=two logId=0000000000000002 parentLogId=0000000000000001`)

		childLogger.InfoContext(ctx, "info", "a", []slog.Attr{slog.Int("i", 1)})
		check(t, `level=INFO msg=info a.i=1 logId=0000000000000002 parentLogId=0000000000000001`)

		childLogger.InfoContext(ctx, "info", "a", slog.GroupValue(slog.Int("i", 1)))
		check(t, `level=INFO msg=info a.i=1 logId=0000000000000002 parentLogId=0000000000000001`)
	})

	t.Run("custom_attr", func(t *testing.T) {
		testutil.SetIDGen(t)

		type ctxKey struct{}
		cslog.AddContextAttrs(cslog.Context("ctxAttr", nil, cslog.GetStringFn(ctxKey{})))

		ctx, logger := cslog.NewLoggerWithContext(
			context.WithValue(
				context.Background(),
				ctxKey{}, "testValue",
			),
		)

		logger.Error("value by logger", "a", 1)
		check(t, `level=ERROR msg="value by logger" a=1 logId=0000000000000000 ctxAttr=testValue`)

		ctx = context.WithValue(ctx, ctxKey{}, "overwritten")
		logger.ErrorContext(ctx, "value is overwittern by context", "a", 1)
		check(t, `level=ERROR msg="value is overwittern by context" a=1 logId=0000000000000000 ctxAttr=overwritten`)
	})
}
