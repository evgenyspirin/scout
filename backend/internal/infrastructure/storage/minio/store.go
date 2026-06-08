// Package miniostore implements object storage backed by MinIO/S3.
// It satisfies photoapp.ObjectStorage and thumbapp.OriginalStore.
package miniostore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Store wraps a MinIO client and a bucket.
type Store struct {
	client *minio.Client
	bucket string
}

// Config configures the MinIO client.
type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// New builds a Store and ensures the bucket exists.
func New(ctx context.Context, cfg Config) (*Store, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("bucket exists: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("make bucket: %w", err)
		}
	}
	return &Store{client: client, bucket: cfg.Bucket}, nil
}

// PresignPut returns a presigned PUT URL for uploading object bytes.
func (s *Store) PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, map[string]string, time.Time, error) {
	u, err := s.client.PresignedPutObject(ctx, s.bucket, key, ttl)
	if err != nil {
		return "", nil, time.Time{}, fmt.Errorf("presign put: %w", err)
	}
	headers := map[string]string{}
	if contentType != "" {
		headers["Content-Type"] = contentType
	}
	return u.String(), headers, time.Now().Add(ttl), nil
}

// ObjectExists reports whether the object is present in the bucket.
func (s *Store) ObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		resp := minio.ToErrorResponse(err)
		if resp.Code == "NoSuchKey" || resp.StatusCode == 404 {
			return false, nil
		}
		return false, fmt.Errorf("stat object: %w", err)
	}
	return true, nil
}

// GetOriginal reads the full object bytes. found is false when missing.
func (s *Store) GetOriginal(ctx context.Context, key string) ([]byte, bool, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, false, fmt.Errorf("get object: %w", err)
	}
	defer obj.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, obj); err != nil {
		resp := minio.ToErrorResponse(err)
		if resp.Code == "NoSuchKey" || resp.StatusCode == 404 {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read object: %w", err)
	}
	return buf.Bytes(), true, nil
}

// StreamOriginal returns a reader for the object for proxy streaming.
func (s *Store) StreamOriginal(ctx context.Context, key string) (io.ReadCloser, string, int64, bool, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		resp := minio.ToErrorResponse(err)
		if resp.Code == "NoSuchKey" || resp.StatusCode == 404 {
			return nil, "", 0, false, nil
		}
		return nil, "", 0, false, fmt.Errorf("stat object: %w", err)
	}
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", 0, false, fmt.Errorf("get object: %w", err)
	}
	ct := info.ContentType
	if ct == "" {
		ct = "image/jpeg"
	}
	return obj, ct, info.Size, true, nil
}

// PutObject uploads bytes directly (used only by tooling/tests, not the API).
func (s *Store) PutObject(ctx context.Context, key, contentType string, data []byte) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	return err
}
