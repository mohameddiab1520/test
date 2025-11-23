package storage

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Storage handles S3/MinIO object storage operations for builds
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

// UploadLogs uploads build logs to S3
func (s *S3Storage) UploadLogs(ctx context.Context, buildID string, logs []byte) (string, error) {
	objectPath := fmt.Sprintf("builds/%s/logs.txt", buildID)

	reader := bytes.NewReader(logs)
	_, err := s.client.PutObject(ctx, s.bucketName, objectPath, reader, int64(len(logs)),
		minio.PutObjectOptions{ContentType: "text/plain"})
	if err != nil {
		return "", fmt.Errorf("failed to upload logs: %w", err)
	}

	// Generate presigned download URL (valid for 7 days)
	url, err := s.client.PresignedGetObject(ctx, s.bucketName, objectPath, 7*24*time.Hour, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return url.String(), nil
}

// DownloadLogs downloads build logs from S3
func (s *S3Storage) DownloadLogs(ctx context.Context, logsURL string) ([]string, error) {
	// Extract object path from URL
	objectPath := s.extractObjectPath(logsURL)

	object, err := s.client.GetObject(ctx, s.bucketName, objectPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}
	defer object.Close()

	// Read logs line by line
	var lines []string
	scanner := bufio.NewScanner(object)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read logs: %w", err)
	}

	return lines, nil
}

// UploadArtifact uploads a build artifact to S3
func (s *S3Storage) UploadArtifact(ctx context.Context, buildID, artifactPath string) (string, int64, string, error) {
	// Open artifact file
	file, err := os.Open(artifactPath)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to open artifact: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to stat artifact: %w", err)
	}

	// Calculate MD5 checksum
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", 0, "", fmt.Errorf("failed to calculate checksum: %w", err)
	}
	checksum := fmt.Sprintf("%x", hash.Sum(nil))

	// Reset file pointer
	if _, err := file.Seek(0, 0); err != nil {
		return "", 0, "", fmt.Errorf("failed to reset file pointer: %w", err)
	}

	// Upload to S3
	fileName := filepath.Base(artifactPath)
	objectPath := fmt.Sprintf("builds/%s/artifacts/%s", buildID, fileName)

	_, err = s.client.PutObject(ctx, s.bucketName, objectPath, file, fileInfo.Size(),
		minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to upload artifact: %w", err)
	}

	// Generate presigned download URL (valid for 30 days)
	url, err := s.client.PresignedGetObject(ctx, s.bucketName, objectPath, 30*24*time.Hour, nil)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return url.String(), fileInfo.Size(), checksum, nil
}

// DeleteBuildArtifacts deletes all artifacts for a build
func (s *S3Storage) DeleteBuildArtifacts(ctx context.Context, buildID string) error {
	objectPrefix := fmt.Sprintf("builds/%s/", buildID)

	objectsCh := make(chan minio.ObjectInfo)

	// List all objects with prefix
	go func() {
		defer close(objectsCh)
		opts := minio.ListObjectsOptions{
			Prefix:    objectPrefix,
			Recursive: true,
		}
		for object := range s.client.ListObjects(ctx, s.bucketName, opts) {
			if object.Err != nil {
				return
			}
			objectsCh <- object
		}
	}()

	// Delete all objects
	errorCh := s.client.RemoveObjects(ctx, s.bucketName, objectsCh, minio.RemoveObjectsOptions{})
	for e := range errorCh {
		if e.Err != nil {
			return fmt.Errorf("failed to delete object %s: %w", e.ObjectName, e.Err)
		}
	}

	return nil
}

// extractObjectPath extracts object path from presigned URL
func (s *S3Storage) extractObjectPath(url string) string {
	// Simple extraction - in production, use proper URL parsing
	// This assumes URL format: http(s)://endpoint/bucket/path?query
	// Returns: builds/xxx/logs.txt
	return filepath.Base(url)
}
