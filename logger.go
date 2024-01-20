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
		logger *Logger
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
	return DefaultProvider().logger
}

// SetInnerHandler sets the default logger provider's handler.
func SetInnerHandler(handler slog.Handler) {
	defaultLoggerProvider.SetInnerHandler(handler)
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
func NewLoggerProvider(innerHandler slog.Handler) *LoggerProvider {
	handler := NewContextHandler(innerHandler).WithContextAttrs(
		Context(keyLogId, nil, getLogIdFunc, nil),
		Context(keyParentLogId, nil, getParentLogIdFunc, nil),
	)

	return &LoggerProvider{
		logger: newLogger(handler),
	}
}

// SetInnerHandler sets the handler.
func (p *LoggerProvider) SetInnerHandler(handler slog.Handler) {
	p.logger.contextHandler().SetInnerHandler(handler)
}

// AddContextAttrs sets the attr (key-value pair) obtained from context to be output to the log.
// See also [ContextAttr].
func (p *LoggerProvider) AddContextAttrs(attrs ...ContextAttr) {
	p.logger = p.logger.WithContextAttrs(attrs...)
}

// NewLogger returns Logger.
func (p *LoggerProvider) NewLogger() *Logger {
	return newLogger(p.logger.contextHandler().clone())
}

// NewLoggerWithContext returns a context and a logger by [Logger.WithContext]
func (p *LoggerProvider) NewLoggerWithContextAttrs(attrs ...ContextAttr) *Logger {
	return p.NewLogger().WithContextAttrs(attrs...)
}

// NewLoggerWithContextAttrs returns a context and a logger by [Logger.WithContextAttrs]
func (p *LoggerProvider) NewLoggerWithContext(ctx context.Context) (context.Context, *Logger) {
	return p.logger.WithContext(ctx)
}

// NewLoggerWithChildContext returns a context and a logger by [Logger.WithChildContext]
func (p *LoggerProvider) NewLoggerWithChildContext(ctx context.Context) (context.Context, *Logger) {
	return p.logger.WithChildContext(ctx)
}

// AddContextAttrs calls [LoggerProvider.AddContextAttrs] on the default provider.
func AddContextAttrs(attrs ...ContextAttr) {
	DefaultProvider().AddContextAttrs(attrs...)
}

// NewLoggerWithContextAttrs calls [LoggerProvider.NewLoggerWithContextAttrs] on the default provider.
func NewLoggerWithContextAttrs(attrs ...ContextAttr) *Logger {
	return DefaultProvider().NewLoggerWithContextAttrs(attrs...)
}

// NewLoggerWithContext calls [LoggerProvider.NewLoggerWithContext] on the default provider.
func NewLoggerWithContext(ctx context.Context) (context.Context, *Logger) {
	return DefaultProvider().NewLoggerWithContext(ctx)
}

// NewContextLogger calls [LoggerProvider.NewLoggerWithChildContext] on the default provider.
func NewLoggerWithChildContext(ctx context.Context) (context.Context, *Logger) {
	return DefaultProvider().NewLoggerWithChildContext(ctx)
}

func (l *Logger) clone() *Logger {
	c := *l
	return &c
}

func (l *Logger) Handler() slog.Handler {
	return l.sl.Handler()
}

func (l *Logger) contextHandler() *ContextHandler {
	if ctxHandler, ok := l.Handler().(*ContextHandler); ok {
		return ctxHandler
	}
	panic("invalid Handler")
}

// newLogger returns Logger.
func newLogger(h *ContextHandler) *Logger {
	if h == nil {
		panic("nil Handler")
	}
	return &Logger{
		sl: slog.New(h),
	}
}

// NewLogger returns Logger.
func NewLogger(innerHandler slog.Handler) *Logger {
	return NewLoggerProvider(innerHandler).NewLogger()
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

// WithContextAttrs returns a Logger that includes the given context
// attributes in each output operation.
func (l *Logger) WithContextAttrs(attrs ...ContextAttr) *Logger {
	return newLogger(l.contextHandler().WithContextAttrs(attrs...))
}

// setContextAttrs returns a Logger that includes the given context
// attributes in each output operation.
// The old context attributes is replaced by the given attrs.
func (l *Logger) setContextAttrs(attrs ...ContextAttr) *Logger {
	return newLogger(l.contextHandler().SetContextAttrs(attrs))
}

// NewLoggerWithContext creates a new context and a corresponding logger.
// If the provided context (ctx) does not have a logId, a new logId is generated, and it is set to the context.
// The created logger includes the logId, parentLogId, and other context attributes set in the provider based on the context.
// The context attributes' default values are set to the values found in the given context, if they exist.
func (l *Logger) WithContext(ctx context.Context) (context.Context, *Logger) {
	newCtx := ctx
	newAttrs := []ContextAttr{}

	// Set logId
	if id := GetLogID(ctx); id == nil || id.IsZero() {
		newCtx = WithLogContext(ctx)
	}
	logId := GetLogID(newCtx).String()

	newAttrs = append(newAttrs, Context(
		keyLogId,
		logId,
		getLogIdFunc,
		nil,
	))

	// Set parentLogId
	var parentId any
	if pid := GetParentLogID(ctx); pid != nil && !pid.IsZero() {
		parentId = pid.String()
	}
	newAttrs = append(newAttrs, Context(
		keyParentLogId,
		parentId,
		getParentLogIdFunc,
		nil,
	))

	// Set attrs handler already has.
	for _, attr := range l.contextHandler().attrs {
		if attr.key == keyLogId || attr.key == keyParentLogId {
			continue
		}

		var defaultValue any
		if currentValue, ok := attr.getFn(ctx); ok {
			defaultValue = currentValue
		}

		newAttrs = append(newAttrs, Context(
			attr.key,
			defaultValue, // use current context's value as default value.
			attr.getFn,
			nil,
		))
	}

	newLogger := l.setContextAttrs(newAttrs...)
	return newCtx, newLogger
}

// NewLoggerWithChildContext creates a new child context and a corresponding logger.
// If the provided context (ctx) has an existing logId, the parentLogId is set to the logId,
// and it generates a new logId.
// The child context includes both parentLogId and logId, and a new logger is created based on this child context.
func (l *Logger) WithChildContext(ctx context.Context) (context.Context, *Logger) {
	return l.WithContext(WithChildLogContext(ctx))
}

func (l *Logger) Enabled(ctx context.Context, level slog.Level) bool {
	return l.sl.Enabled(ctx, level)
}

func (l *Logger) Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	l.HandleLog(ctx, level, 0, msg, args...)
}

func (l *Logger) LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	l.HandleLogAttrs(ctx, level, 0, msg, attrs...)
}

func (l *Logger) Debug(msg string, args ...any) {
	l.HandleLog(context.Background(), slog.LevelDebug, 0, msg, args...)
}

func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.HandleLog(ctx, slog.LevelDebug, 0, msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.HandleLog(context.Background(), slog.LevelInfo, 0, msg, args...)
}

func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.HandleLog(ctx, slog.LevelInfo, 0, msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.HandleLog(context.Background(), slog.LevelWarn, 0, msg, args...)
}

func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.HandleLog(ctx, slog.LevelWarn, 0, msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.HandleLog(context.Background(), slog.LevelError, 0, msg, args...)
}

func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.HandleLog(ctx, slog.LevelError, 0, msg, args...)
}

// HandleLog is the low-level logging method for methods that take ...any.
// When it is called directly by a logging method or function, set callDepth is 0.
// If you create a wrapped logging function and want to include source code locations in the log,
// set the appropriate callDepth.
func (l *Logger) HandleLog(ctx context.Context, level slog.Level, callDepth int, msg string, args ...any) {
	if !l.Enabled(ctx, level) {
		return
	}
	var pc uintptr
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(3+callDepth, pcs[:])
	pc = pcs[0]

	r := slog.NewRecord(now(), level, msg, pc)
	r.Add(args...)
	if ctx == nil {
		ctx = context.Background()
	}
	_ = l.Handler().Handle(ctx, r)
}

// HandleLogAttrs is like [Logger.log], but for methods that take ...Attr.
func (l *Logger) HandleLogAttrs(ctx context.Context, level slog.Level, callDepth int, msg string, attrs ...slog.Attr) {
	if !l.Enabled(ctx, level) {
		return
	}
	var pc uintptr
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(3+callDepth, pcs[:])
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
	DefaultLogger().HandleLog(ctx, level, 0, msg, args...)
}

// LogAttrs calls Logger.LogAttrs on the default logger.
func LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	DefaultLogger().HandleLogAttrs(ctx, level, 0, msg, attrs...)
}

// Debug calls Logger.Debug on the default logger.
func Debug(msg string, args ...any) {
	DefaultLogger().HandleLog(context.Background(), slog.LevelDebug, 0, msg, args...)
}

// DebugContext calls Logger.DebugContext on the default logger.
func DebugContext(ctx context.Context, msg string, args ...any) {
	DefaultLogger().HandleLog(ctx, slog.LevelDebug, 0, msg, args...)
}

// Info calls Logger.Info on the default logger.
func Info(msg string, args ...any) {
	DefaultLogger().HandleLog(context.Background(), slog.LevelInfo, 0, msg, args...)
}

// InfoContext calls Logger.InfoContext on the default logger.
func InfoContext(ctx context.Context, msg string, args ...any) {
	DefaultLogger().HandleLog(ctx, slog.LevelInfo, 0, msg, args...)
}

// Warn calls Logger.Warn on the default logger.
func Warn(msg string, args ...any) {
	DefaultLogger().HandleLog(context.Background(), slog.LevelWarn, 0, msg, args...)
}

// WarnContext calls Logger.WarnContext on the default logger.
func WarnContext(ctx context.Context, msg string, args ...any) {
	DefaultLogger().HandleLog(ctx, slog.LevelWarn, 0, msg, args...)
}

// Error calls Logger.Error on the default logger.
func Error(msg string, args ...any) {
	DefaultLogger().HandleLog(context.Background(), slog.LevelError, 0, msg, args...)
}

// ErrorContext calls Logger.ErrorContext on the default logger.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	DefaultLogger().HandleLog(ctx, slog.LevelError, 0, msg, args...)
}
