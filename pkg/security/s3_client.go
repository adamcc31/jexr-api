package security

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Provider represents the S3-compatible storage provider
type S3Provider string

const (
	S3ProviderAWS    S3Provider = "aws"
	S3ProviderWasabi S3Provider = "wasabi"
)

// S3ClientConfig holds configuration for S3-compatible storage
type S3ClientConfig struct {
	Provider        S3Provider
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Bucket          string

	// Wasabi-specific settings
	WasabiEndpoint string // e.g., "s3.ap-southeast-1.wasabisys.com"
}

// WasabiEndpoints maps regions to Wasabi endpoints
var WasabiEndpoints = map[string]string{
	"us-east-1":      "s3.us-east-1.wasabisys.com",
	"us-east-2":      "s3.us-east-2.wasabisys.com",
	"us-west-1":      "s3.us-west-1.wasabisys.com",
	"eu-central-1":   "s3.eu-central-1.wasabisys.com",
	"eu-west-1":      "s3.eu-west-1.wasabisys.com",
	"eu-west-2":      "s3.eu-west-2.wasabisys.com",
	"ap-northeast-1": "s3.ap-northeast-1.wasabisys.com",
	"ap-northeast-2": "s3.ap-northeast-2.wasabisys.com",
	"ap-southeast-1": "s3.ap-southeast-1.wasabisys.com",
	"ap-southeast-2": "s3.ap-southeast-2.wasabisys.com",
}

// NewS3ClientConfigFromEnv creates S3 config from environment variables
func NewS3ClientConfigFromEnv() S3ClientConfig {
	provider := S3ProviderAWS
	if os.Getenv("S3_PROVIDER") == "wasabi" {
		provider = S3ProviderWasabi
	}

	cfg := S3ClientConfig{
		Provider:        provider,
		AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
		Region:          os.Getenv("S3_REGION"),
		Bucket:          os.Getenv("SECURITY_ANCHOR_BUCKET"),
	}

	// For Wasabi, allow custom endpoint override
	if provider == S3ProviderWasabi {
		if endpoint := os.Getenv("WASABI_ENDPOINT"); endpoint != "" {
			cfg.WasabiEndpoint = endpoint
		} else if endpoint, ok := WasabiEndpoints[cfg.Region]; ok {
			cfg.WasabiEndpoint = endpoint
		} else {
			// Default to ap-southeast-1 if region not found
			cfg.WasabiEndpoint = "s3.ap-southeast-1.wasabisys.com"
		}
	}

	return cfg
}

// NewS3Client creates an S3 client with the given config
// Supports both AWS S3 and Wasabi
func NewS3Client(ctx context.Context, cfg S3ClientConfig) (*s3.Client, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with provider-specific options
	var s3Client *s3.Client

	switch cfg.Provider {
	case S3ProviderWasabi:
		// Wasabi requires custom endpoint and path-style addressing
		s3Client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String("https://" + cfg.WasabiEndpoint)
			o.UsePathStyle = true // Wasabi requires path-style
		})
	default:
		// AWS S3 - use default configuration
		s3Client = s3.NewFromConfig(awsCfg)
	}

	return s3Client, nil
}

// CreateWasabiClient is a convenience function for creating a Wasabi client
func CreateWasabiClient(ctx context.Context, accessKey, secretKey, region string) (*s3.Client, error) {
	cfg := S3ClientConfig{
		Provider:        S3ProviderWasabi,
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		Region:          region,
	}

	// Set endpoint based on region
	if endpoint, ok := WasabiEndpoints[region]; ok {
		cfg.WasabiEndpoint = endpoint
	} else {
		return nil, fmt.Errorf("unknown Wasabi region: %s", region)
	}

	return NewS3Client(ctx, cfg)
}

// TestS3Connection tests the S3/Wasabi connection by listing bucket contents
func TestS3Connection(ctx context.Context, client *s3.Client, bucket string) error {
	_, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		MaxKeys: aws.Int32(1), // Just check if we can access
	})
	if err != nil {
		return fmt.Errorf("failed to access bucket %s: %w", bucket, err)
	}
	return nil
}
