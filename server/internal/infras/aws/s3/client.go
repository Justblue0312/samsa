package s3

import (
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	s3transfermanager "github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"

	samsaConfig "github.com/justblue/samsa/config"
)

type uploaderAPI interface {
	UploadObject(ctx context.Context, params *s3transfermanager.UploadObjectInput, optFns ...func(*s3transfermanager.Options)) (*s3transfermanager.UploadObjectOutput, error)
}

type downloaderAPI interface {
	DownloadObject(ctx context.Context, params *s3transfermanager.DownloadObjectInput, optFns ...func(*s3transfermanager.Options)) (*s3transfermanager.DownloadObjectOutput, error)
	GetObject(ctx context.Context, params *s3transfermanager.GetObjectInput, optFns ...func(*s3transfermanager.Options)) (*s3transfermanager.GetObjectOutput, error)
}

type Client struct {
	Client *s3.Client
	Bucket *string

	uploader   uploaderAPI
	downloader downloaderAPI
}

func NewClient(ctx context.Context, c *samsaConfig.Config, bucket string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(c.AWS.AccessKeyID, c.AWS.SecretAccessKey, "")),
		config.WithRegion(c.AWS.Region),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load s3 config")
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(c.AWS.S3EndpointURL)
		o.UsePathStyle = true
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})
	return &Client{
		Client: client,
		Bucket: aws.String(bucket),
	}, nil
}

func (c *Client) UploadObject(ctx context.Context, key string, fileType string, content io.Reader) (string, error) {
	uploader := c.uploader
	if uploader == nil {
		uploader = s3transfermanager.New(c.Client)
	}
	result, err := uploader.UploadObject(ctx, &s3transfermanager.UploadObjectInput{
		Bucket:      c.Bucket,
		Key:         aws.String(key),
		ContentType: aws.String(fileType),
		Body:        content,
	})
	if err != nil {
		return "", err
	}

	if result.Key == nil || *result.Key == "" {
		return "", errors.New("failed to get file key")
	}
	return *result.Key, nil
}

func (c *Client) PresignGetObject(ctx context.Context, key string) (string, error) {
	presignClient := s3.NewPresignClient(c.Client)
	presignResult, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(*c.Bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(5 * 24 * time.Hour)
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to presign put object")
	}
	return presignResult.URL, nil
}

func (c *Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	downloader := c.downloader
	if downloader == nil {
		downloader = s3transfermanager.New(c.Client)
	}
	result, err := downloader.GetObject(ctx, &s3transfermanager.GetObjectInput{
		Bucket: c.Bucket,
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to download object")
	}
	return io.ReadAll(result.Body)
}

func (c *Client) GetObjectStream(ctx context.Context, key string) (io.ReadCloser, error) {
	output, err := c.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: c.Bucket,
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get object")
	}
	return output.Body, nil
}

func (c *Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: c.Bucket,
		Key:    aws.String(key),
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete object")
	}
	return nil
}

func (c *Client) IsConfigured() bool {
	return c != nil && c.Client != nil
}
