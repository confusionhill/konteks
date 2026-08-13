package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/dika/contextkit/backend/internal/config"
	"github.com/dika/contextkit/backend/internal/domain"
)

type s3StorageProvider struct {
	client *s3.Client
	bucket string
}

func NewS3StorageProvider(cfg *config.Config) (domain.StorageProvider, error) {
	var opts []func(*s3config.LoadOptions) error

	// Bounded default region required by AWS SDK
	opts = append(opts, s3config.WithRegion("us-east-1"))

	// Setup custom credentials if provided
	if cfg.S3AccessKey != "" && cfg.S3SecretKey != "" {
		opts = append(opts, s3config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		))
	}

	awsCfg, err := s3config.LoadDefaultConfig(context.TODO(), opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.S3Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
		}
		o.UsePathStyle = true
	})

	return &s3StorageProvider{
		client: client,
		bucket: cfg.S3Bucket,
	}, nil
}

func (s *s3StorageProvider) PutObject(ctx context.Context, key string, data []byte) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload S3 object: %w", err)
	}
	return key, nil
}

func (s *s3StorageProvider) GetObject(ctx context.Context, key string) ([]byte, error) {
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get S3 object: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 object body: %w", err)
	}

	return data, nil
}

func (s *s3StorageProvider) EnsureBucketExists(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		var nsk *types.NoSuchBucket
		var nf *types.NotFound
		errStr := err.Error()
		if errors.As(err, &nsk) || errors.As(err, &nf) || strings.Contains(errStr, "NotFound") || strings.Contains(errStr, "404") || strings.Contains(errStr, "NoSuchBucket") {
			log.Printf("[Storage]: Bucket '%s' not found. Creating it...", s.bucket)
			_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{
				Bucket: aws.String(s.bucket),
			})
			if err != nil {
				return fmt.Errorf("failed to create S3 bucket: %w", err)
			}
			log.Printf("[Storage]: Bucket '%s' created successfully.", s.bucket)
			return nil
		}
		return fmt.Errorf("failed to check S3 bucket: %w", err)
	}
	return nil
}
