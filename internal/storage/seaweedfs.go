package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rodrigomsrocha/code-vault/internal/config"
)

// SeaweedFS wraps the S3 client for SeaweedFS operations
type SeaweedFS struct {
	client     *s3.Client
	bucketName string
}

// NewSeaweedFS creates a new SeaweedFS client using S3 API
func NewSeaweedFS(cfg *config.SeaweedFSConfig) (*SeaweedFS, error) {
	// Create AWS session configured for SeaweedFS
	awscfg, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awscfg, func(o *s3.Options) {
		o.Region = "us-west-2"
		o.BaseEndpoint = &cfg.S3Endpoint
		o.Credentials = credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")
		o.UsePathStyle = true
		o.DisableLogOutputChecksumValidationSkipped = true
	})

	sfs := &SeaweedFS{
		client:     client,
		bucketName: cfg.BucketName,
	}

	// Ensure bucket exists
	if err := sfs.ensureBucket(context.TODO()); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return sfs, nil
}

// ensureBucket creates the bucket if it doesn't exist
func (sfs *SeaweedFS) ensureBucket(ctx context.Context) error {
	// Check if bucket exists
	_, err := sfs.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(sfs.bucketName),
	})

	if err == nil {
		// Bucket exists
		return nil
	}

	// Bucket doesn't exist, create it
	_, err = sfs.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(sfs.bucketName),
	})

	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

// Upload uploads content to SeaweedFS and returns the object key (FID)
func (sfs *SeaweedFS) Upload(ctx context.Context, key string, content []byte) (string, error) {
	_, err := sfs.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(sfs.bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(content),
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload to SeaweedFS: %w", err)
	}

	return key, nil
}

// Download retrieves content from SeaweedFS by key (FID)
func (sfs *SeaweedFS) Download(ctx context.Context, key string) ([]byte, error) {
	result, err := sfs.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(sfs.bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to download from SeaweedFS: %w", err)
	}
	defer result.Body.Close()

	content, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	return content, nil
}

// Delete removes an object from SeaweedFS
func (sfs *SeaweedFS) Delete(ctx context.Context, key string) error {
	_, err := sfs.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(sfs.bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete from SeaweedFS: %w", err)
	}

	return nil
}

// Exists checks if an object exists in SeaweedFS
func (sfs *SeaweedFS) Exists(ctx context.Context, key string) (bool, error) {
	_, err := sfs.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(sfs.bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		// Object doesn't exist
		return false, nil
	}

	return true, nil
}
