package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/kmio11/cslog"
)

type requestIdCtxKey struct{}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the requestId to the context.
		// All the logs in this request scope includes this requestId.
		ctx := context.WithValue(r.Context(), requestIdCtxKey{}, "6c8b715a-dfe3-40bd-8634-40312fa05897")

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
			// When the log handler processes a Record, it call the function you've set here
			// to retrieve a requestId from the context.
			// Alternatively, you can use a utility function and write it as `cslog.GetFn[string](requestIdCtxKey{})`.
			func(ctx context.Context) (value any, ok bool) {
				value, ok = ctx.Value(requestIdCtxKey{}).(string)
				return
			},
			nil,
		),
	)
	// By default, slog.Default().Handler() is used to handle records.
	// You can use any handler with cslog.SetInnerHandler().
	// If you want to use slog.JSONHandler, call cslog.SetJSONHandler.
	cslog.SetJSONHandler(os.Stdout, &slog.HandlerOptions{})

	// Simulate an HTTP request using httptest instead of starting a server.
	httpHandler := loggingMiddleware(http.HandlerFunc(helloWorldHandler))
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080", nil)
	resp := httptest.NewRecorder()
	httpHandler.ServeHTTP(resp, req)
}
