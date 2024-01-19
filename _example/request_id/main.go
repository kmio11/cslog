package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/kmio11/cslog"
)

type requestIdCtxKey struct{}

func WithRequestId(ctx context.Context, requestId string) context.Context {
	return context.WithValue(ctx, requestIdCtxKey{}, requestId)
}

func GetRequestId(ctx context.Context) string {
	if requestId, ok := ctx.Value(requestIdCtxKey{}).(string); ok {
		return requestId
	}
	return ""
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the requestId to the context.
		// All the logs in this request scope includes this requestId.
		ctx := WithRequestId(r.Context(), "6c8b715a-dfe3-40bd-8634-40312fa05897")

		cslog.InfoContext(ctx, "start request")

		next.ServeHTTP(w, r.WithContext(ctx))

		cslog.InfoContext(ctx, "end request", slog.Int("code", 200))
	})
}

func helloWorldHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cslog.InfoContext(ctx, "start hello")
	fmt.Fprintf(w, "Hello World")
	cslog.InfoContext(ctx, "end hello")
}

func main() {
	// Set up the cslog to output the logs with requestId when the given context has requestId.
	cslog.AddContextAttrs(
		cslog.Context(
			"requestId", nil,
			func(ctx context.Context) (value string, ok bool) {
				requestId := GetRequestId(ctx)
				return requestId, requestId != ""
			},
		),
	)

	// Simulate an HTTP request using httptest instead of starting a server.
	handler := loggingMiddleware(http.HandlerFunc(helloWorldHandler))
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
}
