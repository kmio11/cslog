# cslog

Golang logger using slog with context.

This package is a wrapper for the slog logger, adding any attributes obtained from the context.

## Installation

```
$ go get github.com/kmio11/cslog
```

## Examples

### Adding requestId

It is possible to configure the inclusion of any key/value obtained from the context in the logs.  
The following is a simple example of an HTTP server, where the loggingMiddleware sets the request ID in the context.  
Configure cslog to output this request ID in all logs related to that specific request.

```go
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
```

The outputs are:

```
{"time":"2024-01-01T09:55:54.196341788+09:00","level":"INFO","msg":"start request","requestId":"6c8b715a-dfe3-40bd-8634-40312fa05897"}
{"time":"2024-01-01T09:55:54.196397299+09:00","level":"INFO","msg":"start hello","requestId":"6c8b715a-dfe3-40bd-8634-40312fa05897"}
{"time":"2024-01-01T09:55:54.196407281+09:00","level":"INFO","msg":"end hello","requestId":"6c8b715a-dfe3-40bd-8634-40312fa05897"}
{"time":"2024-01-01T09:55:54.196409716+09:00","level":"INFO","msg":"end request","code":200,"requestId":"6c8b715a-dfe3-40bd-8634-40312fa05897"}
```

### Adding logId / parentLogId

```go
package main

import (
	"context"
	"fmt"

	"github.com/kmio11/cslog"
)

func sub(ctx context.Context, i int) {
	cslog.InfoContext(ctx, fmt.Sprintf("start: sub process %d", i))
	// do something
	cslog.InfoContext(ctx, fmt.Sprintf("end  : sub process %d", i))
}

func main() {
	ctx := cslog.WithLogContext(context.Background())

	cslog.InfoContext(ctx, "start: main")

	for i := 0; i < 3; i++ {
		ctx := cslog.WithChildLogContext(ctx)
		sub(ctx, i)
	}

	cslog.InfoContext(ctx, "end  : main")
}
```

The outputs are:

```
2024/01/01 10:01:53 INFO start: main logId=835f149168a5218b
2024/01/01 10:01:53 INFO start: sub process 0 logId=b5fdb8fd38ad42a4 parentLogId=835f149168a5218b
2024/01/01 10:01:53 INFO end  : sub process 0 logId=b5fdb8fd38ad42a4 parentLogId=835f149168a5218b
2024/01/01 10:01:53 INFO start: sub process 1 logId=ec3bbf8fdd12d4d3 parentLogId=835f149168a5218b
2024/01/01 10:01:53 INFO end  : sub process 1 logId=ec3bbf8fdd12d4d3 parentLogId=835f149168a5218b
2024/01/01 10:01:53 INFO start: sub process 2 logId=9ecbb7d29f00cfc9 parentLogId=835f149168a5218b
2024/01/01 10:01:53 INFO end  : sub process 2 logId=9ecbb7d29f00cfc9 parentLogId=835f149168a5218b
2024/01/01 10:01:53 INFO end  : main logId=835f149168a5218b
```

The outputs include `logId` and `parentLogId`.
