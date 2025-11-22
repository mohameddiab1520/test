package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Storage handles S3/MinIO object storage operations
type S3Storage struct {
	client     *minio.Client
	bucketName string
}

// NewS3Storage creates a new S3 storage client
func NewS3Storage(endpoint, accessKey, secretKey, bucketName, region string, useSSL bool) (*S3Storage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: region})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &S3Storage{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// GeneratePresignedUploadURL generates a presigned URL for uploading
func (s *S3Storage) GeneratePresignedUploadURL(ctx context.Context, objectPath string, expiryDuration time.Duration) (string, error) {
	url, err := s.client.PresignedPutObject(ctx, s.bucketName, objectPath, expiryDuration)
	if err != nil {
		return "", fmt.Errorf("failed to generate upload URL: %w", err)
	}
	return url.String(), nil
}

// GeneratePresignedDownloadURL generates a presigned URL for downloading
func (s *S3Storage) GeneratePresignedDownloadURL(ctx context.Context, objectPath string, expiryDuration time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, s.bucketName, objectPath, expiryDuration, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}
	return url.String(), nil
}

// UploadFile uploads a file directly
func (s *S3Storage) UploadFile(ctx context.Context, objectPath string, reader io.Reader, contentType string, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucketName, objectPath, reader, size,
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	return nil
}

// DownloadFile downloads a file
func (s *S3Storage) DownloadFile(ctx context.Context, objectPath string) (*minio.Object, error) {
	object, err := s.client.GetObject(ctx, s.bucketName, objectPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	return object, nil
}

// DeleteFile deletes a file
func (s *S3Storage) DeleteFile(ctx context.Context, objectPath string) error {
	err := s.client.RemoveObject(ctx, s.bucketName, objectPath, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// GetFileInfo gets file metadata
func (s *S3Storage) GetFileInfo(ctx context.Context, objectPath string) (minio.ObjectInfo, error) {
	info, err := s.client.StatObject(ctx, s.bucketName, objectPath, minio.StatObjectOptions{})
	if err != nil {
		return minio.ObjectInfo{}, fmt.Errorf("failed to get file info: %w", err)
	}
	return info, nil
}
