package types

import (
	"context"
	"io"
)

type ContextReader struct {
	Ctx context.Context
	R   io.Reader
}

func (c *ContextReader) Read(p []byte) (int, error) {
	select {
	case <-c.Ctx.Done():
		return 0, c.Ctx.Err()
	default:
		return c.R.Read(p)
	}
}
