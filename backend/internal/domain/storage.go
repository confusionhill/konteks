package domain

import "context"

type StorageProvider interface {
	PutObject(ctx context.Context, key string, data []byte) (string, error)
	GetObject(ctx context.Context, key string) ([]byte, error)
	EnsureBucketExists(ctx context.Context) error
}
