package cslog

import (
	"context"
	"log/slog"
)

var _ slog.Handler = (*ContextHandler)(nil)

type ContextHandler struct {
	ih    slog.Handler
	attrs []ContextAttr
}

func NewContextHandler(sHandler slog.Handler) *ContextHandler {
	return &ContextHandler{
		ih:    sHandler,
		attrs: []ContextAttr{},
	}
}

func (h *ContextHandler) clone() *ContextHandler {
	// the innner handler is shared by the other cloned handlers.
	return &ContextHandler{
		ih:    h.ih,
		attrs: append([]ContextAttr{}, h.attrs...),
	}
}

func (h *ContextHandler) SetInnerHandler(ih slog.Handler) {
	h.ih = ih
}

func (h *ContextHandler) AddContextAttr(attr ContextAttr) {
	h.attrs = append(h.attrs, attr)
}

func (h *ContextHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.ih.Enabled(ctx, l)
}

// Handle processes the given slog.Record within the context.
// It enhances the Record's attributes with the context attributes obtained from the context.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, a := range h.attrs {
		if attr, ok := a.Attr(ctx); ok {
			r.AddAttrs(attr)
		}
	}
	return h.ih.Handle(ctx, r)
}

func (h *ContextHandler) WithAttrs(as []slog.Attr) slog.Handler {
	c := h.clone()
	c.ih = h.ih.WithAttrs(as)
	return c
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	c := h.clone()
	c.ih = h.ih.WithGroup(name)
	return c
}

// SetContextAttrs returns a new Handler with the given context attributes.
// The receiver's existing context attributes are replaced.
func (h *ContextHandler) SetContextAttrs(attrs []ContextAttr) *ContextHandler {
	c := h.clone()
	c.attrs = attrs
	return c
}

// WithContextAttrs returns a new Handler with the given context attributes appended to
// the receiver's existing context attributes.
func (h *ContextHandler) WithContextAttrs(attrs ...ContextAttr) *ContextHandler {
	c := h.clone()
	for _, a := range attrs {
		c.AddContextAttr(a)
	}
	return c
}
