package metadata

import "context"

type NoopReader struct{}

func (n *NoopReader) Read(ctx context.Context, path string) (map[string]any, error) {
	return map[string]any{}, nil
}
