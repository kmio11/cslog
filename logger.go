package cslog

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"
)

const (
	keyLogId       = "logId"
	keyParentLogId = "parentLogId"
)

type (
	LoggerProvider struct {
		handler *ContextHandler
	}

	Logger struct {
		sl *slog.Logger
	}
)

var (
	// logLevel represents the log level of the defaultLoggingProvider.
	// When using a custom handler, this logLevel may be ignored.
	// To utilize this logLevel in your custom handler, retrieve it using the [LogLevel] function.
	logLevel              = new(slog.LevelVar)
	defaultLoggerProvider = newDefaultProvider(os.Stdout)

	// NowFunc returns current time.
	// This function is exported for testing.
	NowFunc = func() time.Time {
		return time.Now()
	}
)

func now() time.Time {
	return NowFunc()
}

func newDefaultProvider(w io.Writer) *LoggerProvider {
	return NewLoggerProvider(
		slog.NewTextHandler(w, &slog.HandlerOptions{
			Level:       logLevel,
			AddSource:   false,
			ReplaceAttr: nil,
		}),
	)
}

// DefaultProvider returns the default logger provider.
func DefaultProvider() *LoggerProvider {
	return defaultLoggerProvider
}

// DefaultProvider returns the logger provided by the default logger provider.
func DefaultLogger() *Logger {
	return DefaultProvider().NewLogger()
}

// SetHandler sets the default logger provider's handler.
func SetHandler(handler slog.Handler) {
	defaultLoggerProvider = NewLoggerProvider(handler)
}

// SetLogLevel sets the log level of the default logger provider.
func SetLogLevel(level slog.Level) {
	logLevel.Set(level)
}

// LogLevel returns log level of the default logger provider.
func LogLevel() *slog.LevelVar {
	return logLevel
}

// NewLoggerProvider returns LoggerProvider.
func NewLoggerProvider(handler slog.Handler) *LoggerProvider {
	return &LoggerProvider{
		handler: NewContextHandler(handler),
	}
}

// NewLogger returns Logger.
func (p *LoggerProvider) NewLogger() *Logger {
	return NewLogger(p.handler)
}

// GetContextLogger creates a new context and a corresponding logger.
// If the provided context (ctx) does not have a logId, a new logId is generated.
// The created logger includes the logId and parentLogId attributes based on the context.
func (p *LoggerProvider) GetContextLogger(ctx context.Context) (context.Context, *Logger) {
	newCtx := ctx
	logId := GetLogID(ctx)
	if logId.IsZero() {
		newCtx = WithLogContext(ctx)
	}
	newLogger := p.NewLogger().With(
		keyLogId, GetLogID(newCtx),
		keyParentLogId, GetParentLogID(ctx),
	)

	return newCtx, newLogger
}

// CreateChildContextLogger creates a new child context and a corresponding logger.
// If the provided context (ctx) has an existing logId, the parentLogId is set to the logId,
// and it generates a new logId.
// The child context includes both parentLogId and logId, and a new logger is created based on this child context.
func (p *LoggerProvider) CreateChildContextLogger(ctx context.Context) (context.Context, *Logger) {
	return p.GetContextLogger(WithChildLogContext(ctx))
}

// GetContextLogger calls LoggerProvider.GetContextLogger on the default provider.
// See [LoggerProvider.GetContextLogger].
func GetContextLogger(ctx context.Context) (context.Context, *Logger) {
	return DefaultProvider().GetContextLogger(ctx)
}

// NewContextLogger calls LoggerProvider.CreateChildContextLogger on the default provider.
// See [LoggerProvider.CreateChildContextLogger].
func CreateChildContextLogger(ctx context.Context) (context.Context, *Logger) {
	return DefaultProvider().CreateChildContextLogger(ctx)
}

func (l *Logger) clone() *Logger {
	c := *l
	return &c
}

func (l *Logger) Handler() slog.Handler {
	return l.sl.Handler()
}

func (l *Logger) With(args ...any) *Logger {
	c := l.clone()
	c.sl = l.sl.With(args...)
	return c
}

func (l *Logger) WithGroup(name string) *Logger {
	c := l.clone()
	c.sl = l.sl.WithGroup(name)
	return c
}

func NewLogger(h slog.Handler) *Logger {
	if h == nil {
		panic("nil Handler")
	}
	return &Logger{
		sl: slog.New(h),
	}
}

func (l *Logger) Enabled(ctx context.Context, level slog.Level) bool {
	return l.sl.Enabled(ctx, level)
}

func (l *Logger) Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	l.log(ctx, level, msg, args...)
}

func (l *Logger) LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	l.logAttrs(ctx, level, msg, attrs...)
}

func (l *Logger) Debug(msg string, args ...any) {
	l.log(context.Background(), slog.LevelDebug, msg, args...)
}

func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, slog.LevelDebug, msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.log(context.Background(), slog.LevelInfo, msg, args...)
}

func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, slog.LevelInfo, msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.log(context.Background(), slog.LevelWarn, msg, args...)
}

func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, slog.LevelWarn, msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.log(context.Background(), slog.LevelError, msg, args...)
}

func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.log(ctx, slog.LevelError, msg, args...)
}

// log is the low-level logging method for methods that take ...any.
// It must always be called directly by an exported logging method
// or function, because it uses a fixed call depth to obtain the pc.
func (l *Logger) log(ctx context.Context, level slog.Level, msg string, args ...any) {
	if !l.Enabled(ctx, level) {
		return
	}
	var pc uintptr
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(3, pcs[:])
	pc = pcs[0]

	r := slog.NewRecord(now(), level, msg, pc)
	r.Add(args...)
	if ctx == nil {
		ctx = context.Background()
	}
	_ = l.Handler().Handle(ctx, r)
}

// logAttrs is like [Logger.log], but for methods that take ...Attr.
func (l *Logger) logAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	if !l.Enabled(ctx, level) {
		return
	}
	var pc uintptr
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(3, pcs[:])
	pc = pcs[0]

	r := slog.NewRecord(now(), level, msg, pc)
	r.AddAttrs(attrs...)
	if ctx == nil {
		ctx = context.Background()
	}
	_ = l.Handler().Handle(ctx, r)
}

// Log calls Logger.Log on the default logger.
func Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	DefaultLogger().log(ctx, level, msg, args...)
}

// LogAttrs calls Logger.LogAttrs on the default logger.
func LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	DefaultLogger().logAttrs(ctx, level, msg, attrs...)
}

// DebugContext calls Logger.DebugContext on the default logger.
func DebugContext(ctx context.Context, msg string, args ...any) {
	DefaultLogger().log(ctx, slog.LevelDebug, msg, args...)
}

// InfoContext calls Logger.InfoContext on the default logger.
func InfoContext(ctx context.Context, msg string, args ...any) {
	DefaultLogger().log(ctx, slog.LevelInfo, msg, args...)
}

// WarnContext calls Logger.WarnContext on the default logger.
func WarnContext(ctx context.Context, msg string, args ...any) {
	DefaultLogger().log(ctx, slog.LevelWarn, msg, args...)
}

// ErrorContext calls Logger.ErrorContext on the default logger.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	DefaultLogger().log(ctx, slog.LevelError, msg, args...)
}
