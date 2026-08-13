package domain

import (
	"context"
	"time"
)
type StorageProvider interface {
	PutObject(ctx context.Context, key string, data []byte) (string, error)
	GetObject(ctx context.Context, key string) ([]byte, error)
	EnsureBucketExists(ctx context.Context) error
	GeneratePresignedURL(ctx context.Context, key string, lifetime time.Duration) (string, error)
}
