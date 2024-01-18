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
