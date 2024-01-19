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
		Context(keyLogId, nil, getLogIdFunc),
		Context(keyParentLogId, nil, getParentLogIdFunc),
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
	p.logger.contextHandler().AddContextAttrs(attrs...)
}

// NewLogger returns Logger.
func (p *LoggerProvider) NewLogger() *Logger {
	return newLogger(p.logger.contextHandler())
}

// NewLoggerWithContext returns a context and a logger by [Logger.WithContext]
func (p *LoggerProvider) NewLoggerWithContextAttrs(overwrite bool, attrs ...ContextAttr) *Logger {
	return p.NewLogger().WithContextAttrs(overwrite, attrs...)
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
func NewLoggerWithContextAttrs(overwrite bool, attrs ...ContextAttr) *Logger {
	return DefaultProvider().NewLoggerWithContextAttrs(overwrite, attrs...)
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
// If overwrite is true, the context attributes of the new handler are replaced by the given attrs.
// Otherwise, the given attrs are added to the existing context attributes of the new handler.
func (l *Logger) WithContextAttrs(overwrite bool, attrs ...ContextAttr) *Logger {
	oldHandler := l.contextHandler()

	var newHandler *ContextHandler
	if overwrite {
		newHandler = oldHandler.SetContextAttrs(attrs)
	} else {
		newHandler = oldHandler.WithContextAttrs(attrs...)
	}

	return newLogger(newHandler)
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
		&logId,
		getLogIdFunc,
	))

	// Set parentLogId
	var parentId *string
	if pid := GetParentLogID(ctx); pid != nil && !pid.IsZero() {
		ppid := pid.String()
		parentId = &ppid
	}
	newAttrs = append(newAttrs, Context(
		keyParentLogId,
		parentId,
		getParentLogIdFunc,
	))

	// Set attrs handler already has.
	for _, attr := range l.contextHandler().attrs {
		if attr.key == keyLogId || attr.key == keyParentLogId {
			continue
		}

		var defaultValue *string
		if currentValue, ok := attr.getFn(ctx); ok {
			defaultValue = &currentValue
		}

		newAttrs = append(newAttrs, Context(
			attr.key,
			defaultValue, // use current context's value as default value.
			attr.getFn,
		))
	}

	newLogger := l.WithContextAttrs(true, newAttrs...)
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

// Debug calls Logger.Debug on the default logger.
func Debug(msg string, args ...any) {
	DefaultLogger().log(context.Background(), slog.LevelDebug, msg, args...)
}

// DebugContext calls Logger.DebugContext on the default logger.
func DebugContext(ctx context.Context, msg string, args ...any) {
	DefaultLogger().log(ctx, slog.LevelDebug, msg, args...)
}

// Info calls Logger.Info on the default logger.
func Info(msg string, args ...any) {
	DefaultLogger().log(context.Background(), slog.LevelInfo, msg, args...)
}

// InfoContext calls Logger.InfoContext on the default logger.
func InfoContext(ctx context.Context, msg string, args ...any) {
	DefaultLogger().log(ctx, slog.LevelInfo, msg, args...)
}

// Warn calls Logger.Warn on the default logger.
func Warn(msg string, args ...any) {
	DefaultLogger().log(context.Background(), slog.LevelWarn, msg, args...)
}

// WarnContext calls Logger.WarnContext on the default logger.
func WarnContext(ctx context.Context, msg string, args ...any) {
	DefaultLogger().log(ctx, slog.LevelWarn, msg, args...)
}

// Error calls Logger.Error on the default logger.
func Error(msg string, args ...any) {
	DefaultLogger().log(context.Background(), slog.LevelError, msg, args...)
}

// ErrorContext calls Logger.ErrorContext on the default logger.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	DefaultLogger().log(ctx, slog.LevelError, msg, args...)
}
