package modules

import (
	"context"
	"io"
	"os"
)

type outputKey struct{}

// WithOutput carries the run's writer to modules.
func WithOutput(ctx context.Context, out io.Writer) context.Context {
	return context.WithValue(ctx, outputKey{}, out)
}

func outputFromContext(ctx context.Context) io.Writer {
	if out, ok := ctx.Value(outputKey{}).(io.Writer); ok {
		return out
	}
	return os.Stdout
}
