package cslog

import (
	"context"
	"log/slog"
)

var _ slog.Handler = (*ContextHandler)(nil)

type ContextHandler struct {
	ih slog.Handler

	logId       LogID
	parentLogId LogID
}

func NewContextHandler(sHandler slog.Handler) *ContextHandler {
	return &ContextHandler{
		ih: sHandler,
	}
}

func (h *ContextHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.ih.Enabled(ctx, l)
}

// Handle processes the given slog.Record within the context.
// It enhances the Record's attributes with logId and parentLogId obtained from the ContextHandler.
// If the provided context (ctx) contains logId or parentId, it uses those values.
// If the value of logId or parentLogId is zero, it is ignored.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	logId := h.logId
	if ctxLogId := GetLogID(ctx); !ctxLogId.IsZero() {
		logId = ctxLogId
	}

	parentLogId := h.parentLogId
	if ctxParentLogId := GetParentLogID(ctx); !ctxParentLogId.IsZero() {
		parentLogId = ctxParentLogId
	}

	if !logId.IsZero() {
		r.AddAttrs(
			slog.String(keyLogId, logId.String()),
		)
	}

	if !parentLogId.IsZero() {
		r.AddAttrs(
			slog.String(keyParentLogId, parentLogId.String()),
		)
	}

	return h.ih.Handle(ctx, r)
}

func (h *ContextHandler) WithAttrs(as []slog.Attr) slog.Handler {
	c := &ContextHandler{}

	filteredAs := []slog.Attr{}
	for _, a := range as {
		// Store logId and parentLogId in ContextHandler's properties instead of Handler.WithAttrs.
		// This is done to avoid duplicate output of logId/parentLogId by Handler.WithAttrs
		// and Record.AddAttrs in ContextHandler.Handle.
		if a.Key == keyLogId {
			c.logId = getLogIdFromAttrValue(a.Value)
			continue
		}
		if a.Key == keyParentLogId {
			c.parentLogId = getLogIdFromAttrValue(a.Value)
			continue
		}
		filteredAs = append(filteredAs, a)
	}

	c.ih = h.ih.WithAttrs(filteredAs)

	return c
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{
		ih: h.ih.WithGroup(name),
	}
}

func getLogIdFromAttrValue(v slog.Value) LogID {
	if vv, ok := v.Any().(LogID); ok {
		return vv
	}
	return Nil
}
