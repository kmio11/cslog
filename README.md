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
```

The outputs are:

```
time=2024-01-01T12:01:39.654+09:00 level=INFO msg="start request" requestId=6c8b715a-dfe3-40bd-8634-40312fa05897
time=2024-01-01T12:01:39.655+09:00 level=INFO msg="start hello" requestId=6c8b715a-dfe3-40bd-8634-40312fa05897
time=2024-01-01T12:01:39.655+09:00 level=INFO msg="end hello" requestId=6c8b715a-dfe3-40bd-8634-40312fa05897
time=2024-01-01T12:01:39.655+09:00 level=INFO msg="end request" code=200 requestId=6c8b715a-dfe3-40bd-8634-40312fa05897
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
time=2024-01-01T16:26:52.392+09:00 level=INFO msg="start: main" logId=d0bbfc94ea303585
time=2024-01-01T16:26:52.392+09:00 level=INFO msg="start: sub process 0" logId=c843a7e179ee6657 parentLogId=d0bbfc94ea303585
time=2024-01-01T16:26:52.392+09:00 level=INFO msg="end  : sub process 0" logId=c843a7e179ee6657 parentLogId=d0bbfc94ea303585
time=2024-01-01T16:26:52.392+09:00 level=INFO msg="start: sub process 1" logId=1bc44979ec589d87 parentLogId=d0bbfc94ea303585
time=2024-01-01T16:26:52.392+09:00 level=INFO msg="end  : sub process 1" logId=1bc44979ec589d87 parentLogId=d0bbfc94ea303585
time=2024-01-01T16:26:52.392+09:00 level=INFO msg="start: sub process 2" logId=62fae7f4e90323b5 parentLogId=d0bbfc94ea303585
time=2024-01-01T16:26:52.392+09:00 level=INFO msg="end  : sub process 2" logId=62fae7f4e90323b5 parentLogId=d0bbfc94ea303585
time=2024-01-01T16:26:52.392+09:00 level=INFO msg="end  : main" logId=d0bbfc94ea303585
```

The outputs include `logId` and `parentLogId`.
