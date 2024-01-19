package cslog

import (
	"context"
	"log/slog"
)

// ContextAttr represents an attribute obtained from the context.
//   - key: The key to be included in the log output.
//   - defaultValue: If not nil, defaultValue is used in the log when the value cannot be obtained from the context using getFn.
//   - getFn: A function to retrieve the value from the context. The returned string is included in the log.
//     If ok is false, the defaultValue is used. If defaultValue is nil and ok is false,
//     the key-value pair is omitted from the log.
type ContextAttr struct {
	key          string
	defaultValue *string
	getFn        func(ctx context.Context) (value string, ok bool)
}

// Context returns an ContextAttr.
func Context(
	key string,
	defaultValue *string,
	getFn func(ctx context.Context) (value string, ok bool),
) ContextAttr {
	return ContextAttr{
		key:          key,
		defaultValue: defaultValue,
		getFn:        getFn,
	}
}

func (a ContextAttr) Attr(ctx context.Context) (slog.Attr, bool) {
	value := a.defaultValue

	if v, ok := a.getFn(ctx); ok {
		value = &v
	}

	if value == nil {
		return slog.Attr{}, false
	}

	return slog.String(a.key, *value), true
}
