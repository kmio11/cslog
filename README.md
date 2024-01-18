# cslog

Golang logger using slog with context.

This package is a wrapper for the slog logger, adding identifiers for the current span and the parent span obtained from the context.

## Installation

```
$ go get github.com/kmio11/cslog
```

## Example

```go
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
