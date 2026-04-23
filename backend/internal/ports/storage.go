package ports

import (
	"context"
	"io"
	"time"
)

// Storage is the port for object storage (Cloudflare R2 / S3-compatible).
type Storage interface {
	Put(ctx context.Context, key string, body io.Reader, contentType string) error
	PresignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}
