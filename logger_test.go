package cslog_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/kmio11/cslog"
	"github.com/kmio11/cslog/testutil"
)

// textTimeRE is a regexp to match log timestamps for Text handler.
// This is RFC3339Nano with the fixed 3 digit sub-second precision.
const textTimeRE = `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}(Z|[+-]\d{2}:\d{2})`

func TestLog(t *testing.T) {

	h := testutil.NewBufTextHandler(t, testutil.BufHandlerOpts{})
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
		t.Cleanup(h.SetLevel(t, slog.LevelDebug))

		ctx := cslog.WithLogContext(context.Background())

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
		t.Cleanup(h.SetLevel(t, slog.LevelDebug))

		_ = cslog.WithLogContext(context.Background())

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
		t.Cleanup(h.SetLevel(t, slog.LevelDebug))

		ctx := cslog.WithLogContext(context.Background())
		childCtx := cslog.WithChildLogContext(ctx)

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
		t.Cleanup(h.SetLevel(t, slog.LevelDebug))

		type ctxKey struct{}
		cslog.AddContextAttrs(
			cslog.Context("ctxAttr", nil, cslog.GetFn[string](ctxKey{}), nil),
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
	h := testutil.NewBufTextHandler(t, testutil.BufHandlerOpts{})
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
		t.Cleanup(h.SetLevel(t, slog.LevelDebug))

		ctx, logger := cslog.NewLoggerWithContext(context.Background())

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
		t.Cleanup(h.SetLevel(t, slog.LevelDebug))

		parentCtx, _ := cslog.NewLoggerWithContext(context.Background())
		ctx, childLogger := cslog.NewLoggerWithChildContext(parentCtx)

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
		t.Cleanup(h.SetLevel(t, slog.LevelDebug))

		parentCtx, _ := cslog.NewLoggerWithContext(context.Background())
		ctx, childLogger := cslog.NewLoggerWithChildContext(parentCtx)

		// If ctx has logId/parentLogId, logger's logId/parentLogId is overwritten.
		ctx = cslog.WithChildLogContext(ctx)

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
		t.Cleanup(h.SetLevel(t, slog.LevelDebug))

		type ctxKey struct{}
		cslog.AddContextAttrs(cslog.Context("ctxAttr", nil, cslog.GetFn[string](ctxKey{}), nil))

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

// callerPC returns the program counter at the given stack depth.
func callerPC(depth int) uintptr {
	var pcs [1]uintptr
	runtime.Callers(depth, pcs[:])
	return pcs[0]
}

func TestCallDepth(t *testing.T) {
	h := testutil.NewBufJSONHandler(t, testutil.BufHandlerOpts{
		AddSource: true,
	})
	cslog.SetInnerHandler(h)
	t.Cleanup(h.SetLevel(t, slog.LevelDebug))

	check := func(wantLine int) {
		t.Helper()
		defer h.ResetBuf(t)

		gotMessage := h.Object(t)["msg"].(string)

		got := testutil.TypedJSONObject[slog.Source](t, h.Object(t)["source"])
		gotFile := filepath.Base(got.File)

		const wantFunc = "github.com/kmio11/cslog_test.TestCallDepth"
		const wantFile = "logger_test.go"

		if got.Function != wantFunc || gotFile != wantFile || got.Line != wantLine {
			t.Errorf("%s : got (%s, %s, %d), want (%s, %s, %d)",
				gotMessage,
				got.Function, gotFile, got.Line, wantFunc, wantFile, wantLine)
		}
	}

	ctx, logger := cslog.NewLoggerWithContext(context.Background())

	// Calls to check must be one line apart.
	// Determine line where calls start.
	f, _ := runtime.CallersFrames([]uintptr{callerPC(2)}).Next()
	baseLine := f.Line
	var startLine int
	// Do not change the number of lines between here and the startLines.

	// test logger.Xx
	startLine = baseLine + 5 + 2
	logger.Debug("logger.Debug")
	check(startLine + 0)
	logger.DebugContext(ctx, "logger.DebugContext")
	check(startLine + 2)
	logger.Info("logger.Info")
	check(startLine + 4)
	logger.InfoContext(ctx, "logger.InfoContext")
	check(startLine + 6)
	logger.Warn("logger.Warn")
	check(startLine + 8)
	logger.WarnContext(ctx, "logger.WarnContext")
	check(startLine + 10)
	logger.Error("logger.Error")
	check(startLine + 12)
	logger.ErrorContext(ctx, "logger.ErrorContext")
	check(startLine + 14)
	logger.Log(ctx, slog.LevelError, "logger.Log")
	check(startLine + 16)
	logger.LogAttrs(ctx, slog.LevelError, "logger.LogAttrs")
	check(startLine + 18)

	// test cslog.Xx
	startLine = baseLine + 28 + 2
	logger.Debug("cslog.Debug")
	check(startLine + 0)
	logger.DebugContext(ctx, "cslog.DebugContext")
	check(startLine + 2)
	logger.Info("cslog.Info")
	check(startLine + 4)
	logger.InfoContext(ctx, "cslog.InfoContext")
	check(startLine + 6)
	logger.Warn("cslog.Warn")
	check(startLine + 8)
	logger.WarnContext(ctx, "cslog.WarnContext")
	check(startLine + 10)
	logger.Error("cslog.Error")
	check(startLine + 12)
	logger.ErrorContext(ctx, "cslog.ErrorContext")
	check(startLine + 14)
	logger.Log(ctx, slog.LevelError, "cslog.Log")
	check(startLine + 16)
	logger.LogAttrs(ctx, slog.LevelError, "cslog.LogAttrs")
	check(startLine + 18)
}

// Infof is an example of a user-defined logging function that wraps cslog.
// The log record contains the source position of the caller of infof.
func infof(logger *cslog.Logger, format string, args ...any) {
	if !logger.Enabled(context.Background(), slog.LevelInfo) {
		return
	}
	logger.HandleLog(context.Background(), slog.LevelInfo, 0, fmt.Sprintf(format, args...))
}

// innerErrorf is an example of a user-defined logging function that wraps cslog and
// this function is also wrapped by another function.
// Therefore, set 1 as the callDepth for the HandleLog function.
// The log record contains the source position of the caller of innerErrorf's caller.
func innerErrorf(logger *cslog.Logger, format string, args ...any) {
	if !logger.Enabled(context.Background(), slog.LevelError) {
		return
	}
	logger.HandleLog(context.Background(), slog.LevelError, 1, fmt.Sprintf(format, args...))
}

func errorf(logger *cslog.Logger, err error) {
	innerErrorf(logger, "error is %s", err.Error())
}

func innerErrorAttrs(logger *cslog.Logger, msg string, attrs ...slog.Attr) {
	if !logger.Enabled(context.Background(), slog.LevelError) {
		return
	}
	logger.HandleLogAttrs(context.Background(), slog.LevelError, 1, msg, attrs...)
}

func errorAttrs(logger *cslog.Logger, err error) {
	innerErrorAttrs(logger, "an error occured", slog.String("err", err.Error()))
}

func TestCallDepth_Wrapping(t *testing.T) {
	h := testutil.NewBufJSONHandler(t, testutil.BufHandlerOpts{
		AddSource: true,
	})

	check := func(wantLine int) {
		t.Helper()
		defer h.ResetBuf(t)

		got := testutil.TypedJSONObject[slog.Source](t, h.Object(t)["source"])
		gotFile := filepath.Base(got.File)

		const wantFunc = "github.com/kmio11/cslog_test.TestCallDepth_Wrapping"
		const wantFile = "logger_test.go"

		if got.Function != wantFunc || gotFile != wantFile || got.Line != wantLine {
			t.Errorf("got (%s, %s, %d), want (%s, %s, %d)",
				got.Function, gotFile, got.Line, wantFunc, wantFile, wantLine)
		}
	}

	logger := cslog.NewLogger(h)

	// Calls to check must be one line apart.
	// Determine line where calls start.
	f, _ := runtime.CallersFrames([]uintptr{callerPC(2)}).Next()
	baseLine := f.Line
	// Do not change the number of lines between here and the startLines.

	startLine := baseLine + 3 + 2
	infof(logger, "")
	check(startLine + 0)
	errorf(logger, errors.New(""))
	check(startLine + 2)
	errorAttrs(logger, errors.New(""))
	check(startLine + 4)
}

func TestLogger_WithContextAttrs(t *testing.T) {
	h := testutil.NewBufTextHandler(t, testutil.BufHandlerOpts{
		RemoveTime: true,
	})

	type ctxKey1 struct{}

	t.Run("no_config", func(t *testing.T) {
		logger := cslog.NewLogger(h).WithContextAttrs(
			cslog.Context(
				"key1",
				nil,
				nil,
				nil,
			),
		)

		ctx := context.Background()
		ctxWithValue := context.WithValue(ctx, ctxKey1{}, "value1")

		logger.InfoContext(ctx, "message")
		h.Check(t, `^level=INFO msg=message$`)

		logger.InfoContext(ctxWithValue, "message")
		h.Check(t, `^level=INFO msg=message$`)
	})

	t.Run("defaultValue", func(t *testing.T) {
		logger := cslog.NewLogger(h).WithContextAttrs(
			cslog.Context(
				"key1",
				"defaultValue",
				nil,
				nil,
			),
		)

		ctx := context.Background()
		ctxWithValue := context.WithValue(ctx, ctxKey1{}, "value1")

		logger.Info("message")
		h.Check(t, `^level=INFO msg=message key1=defaultValue$`)

		logger.InfoContext(ctx, "message")
		h.Check(t, `^level=INFO msg=message key1=defaultValue$`)

		logger.InfoContext(ctxWithValue, "message")
		h.Check(t, `^level=INFO msg=message key1=defaultValue$`)
	})

	t.Run("getFn", func(t *testing.T) {
		logger := cslog.NewLogger(h).WithContextAttrs(
			cslog.Context(
				"key1",
				nil,
				cslog.GetFn[string](ctxKey1{}),
				nil,
			),
		)

		ctx := context.Background()
		ctxWithValue := context.WithValue(ctx, ctxKey1{}, "value1")
		ctxWithInvalidTypeValue := context.WithValue(ctx, ctxKey1{}, 100)

		logger.Info("message")
		h.Check(t, `^level=INFO msg=message$`)

		logger.InfoContext(ctx, "message")
		h.Check(t, `^level=INFO msg=message$`)

		logger.InfoContext(ctxWithValue, "message")
		h.Check(t, `^level=INFO msg=message key1=value1$`)

		logger.InfoContext(ctxWithInvalidTypeValue, "message")
		h.Check(t, `^level=INFO msg=message$`)
	})

	t.Run("defaultValue_getFn", func(t *testing.T) {
		logger := cslog.NewLogger(h).WithContextAttrs(
			cslog.Context(
				"key1",
				"defaultValue",
				cslog.GetFn[string](ctxKey1{}),
				nil,
			),
		)

		ctx := context.Background()
		ctxWithValue := context.WithValue(ctx, ctxKey1{}, "value1")

		logger.Info("message")
		h.Check(t, `^level=INFO msg=message key1=defaultValue$`)

		logger.InfoContext(ctx, "message")
		h.Check(t, `^level=INFO msg=message key1=defaultValue$`)

		logger.InfoContext(ctxWithValue, "message")
		h.Check(t, `^level=INFO msg=message key1=value1$`)
	})

	t.Run("setFn", func(t *testing.T) {
		logger := cslog.NewLogger(h).WithContextAttrs(
			cslog.Context(
				"key1",
				"defaultValue",
				cslog.GetFn[string](ctxKey1{}),
				func(key string, value any) (attr slog.Attr, ok bool) {
					if s, ok := value.(string); ok {
						if s == "ignored" {
							return slog.Attr{}, false
						}
						if s == "group" {
							return slog.Group(
								"group1",
								slog.String(key, s),
							), true
						}
						return slog.String("key1-custom", fmt.Sprintf("%s-custom", s)), true
					}
					return slog.Attr{}, false

				},
			),
		)

		ctx := context.Background()
		ctxWithValue := context.WithValue(ctx, ctxKey1{}, "value1")
		ctxWithIgnored := context.WithValue(ctx, ctxKey1{}, "ignored")
		ctxWithGroup := context.WithValue(ctx, ctxKey1{}, "group")
		ctxWithNil := context.WithValue(ctx, ctxKey1{}, nil)

		logger.Info("message")
		h.Check(t, `^level=INFO msg=message key1-custom=defaultValue-custom$`)

		logger.InfoContext(ctx, "message")
		h.Check(t, `^level=INFO msg=message key1-custom=defaultValue-custom$`)

		logger.InfoContext(ctxWithValue, "message")
		h.Check(t, `^level=INFO msg=message key1-custom=value1-custom$`)

		logger.InfoContext(ctxWithIgnored, "message")
		h.Check(t, `^level=INFO msg=message$`)

		logger.InfoContext(ctxWithGroup, "message")
		h.Check(t, `^level=INFO msg=message group1.key1=group$`)

		logger.InfoContext(ctxWithNil, "message")
		h.Check(t, `^level=INFO msg=message key1-custom=defaultValue-custom$`)
	})
}
