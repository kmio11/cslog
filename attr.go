package cslog

import (
	"context"
	"log/slog"
)

// ContextAttr represents an attribute obtained from the context.
//   - key: The key to be included in the log output.
//     If key is empty, the key-value pari is omitted from the log.
//   - defaultValue: If not nil, defaultValue is used in the log when the valuecannot be obtained from the context using getFn.
//   - getFn: A function to retrieve the value from the context. The returned value is included in the log.
//     If ok is false, the defaultValue is used. If defaultValue is nil and ok is false,
//     the key-value pair is omitted from the log by default, or the defaultValue is passed to setFn if setFn is provided.
//   - setFn: A function to create slog.Attr. If setFn is nil, slog.Attr is created with key and value (not nil) as-is.
type ContextAttr struct {
	key          string
	defaultValue any
	getFn        func(ctx context.Context) (value any, ok bool)
	setFn        func(key string, value any) (attr slog.Attr, ok bool)
}

// Context returns an [ContextAttr].
func Context(
	key string,
	defaultValue any,
	getFn func(ctx context.Context) (value any, ok bool),
	setFn func(key string, value any) (attr slog.Attr, ok bool),
) ContextAttr {
	return ContextAttr{
		key:          key,
		defaultValue: defaultValue,
		getFn:        getFn,
		setFn:        setFn,
	}
}

// Attr retrieves the attribute from the context and returns it as a slog.Attr.
// If getFn is provided, it attempts to get the value from the context; otherwise, it uses the defaultValue.
// If setFn is provided, it uses setFn to create the slog.Attr with the obtained or default value.
func (a ContextAttr) Attr(ctx context.Context) (slog.Attr, bool) {
	if a.key == "" {
		return slog.Attr{}, false
	}

	value := a.defaultValue
	if a.getFn != nil {
		if v, ok := a.getFn(ctx); ok {
			value = v
		}
	}

	if a.setFn != nil {
		return a.setFn(a.key, value)
	}

	return SetFn()(a.key, value)
}

// P returns a pointer of v.
func P[T any](v T) *T {
	return &v
}

// GetFn returns a [ContextAttr]'s getFn for a value with a given key.
func GetFn[T any](ctxKey any) func(ctx context.Context) (value any, ok bool) {
	return func(ctx context.Context) (value any, ok bool) {
		value, ok = ctx.Value(ctxKey).(T)
		return
	}
}

// SetFn returns a [ContextAttr]'s setFn.
func SetFn() func(key string, value any) (attr slog.Attr, ok bool) {
	return func(key string, value any) (attr slog.Attr, ok bool) {
		if key == "" || value == nil {
			return slog.Attr{}, false
		}
		return slog.Any(key, value), true
	}
}
