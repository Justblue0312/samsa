package s3

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3transfermanager "github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
)

type mockUploader struct {
	output *s3transfermanager.UploadObjectOutput
	err    error
}

func (m *mockUploader) UploadObject(ctx context.Context, params *s3transfermanager.UploadObjectInput, optFns ...func(*s3transfermanager.Options)) (*s3transfermanager.UploadObjectOutput, error) {
	return m.output, m.err
}

type mockDownloader struct {
	output *s3transfermanager.GetObjectOutput
	err    error
}

func (m *mockDownloader) DownloadObject(ctx context.Context, params *s3transfermanager.DownloadObjectInput, optFns ...func(*s3transfermanager.Options)) (*s3transfermanager.DownloadObjectOutput, error) {
	return nil, m.err
}

func (m *mockDownloader) GetObject(ctx context.Context, params *s3transfermanager.GetObjectInput, optFns ...func(*s3transfermanager.Options)) (*s3transfermanager.GetObjectOutput, error) {
	return m.output, m.err
}

func TestClient_UploadObject(t *testing.T) {
	uploader := &mockUploader{
		output: &s3transfermanager.UploadObjectOutput{
			Key: aws.String("test/key.jpg"),
		},
	}
	client := &Client{
		Client:   &s3.Client{},
		Bucket:   aws.String("test-bucket"),
		uploader: uploader,
	}

	key, err := client.UploadObject(context.Background(), "test/key.jpg", "image/jpeg", nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if key != "test/key.jpg" {
		t.Errorf("expected key 'test/key.jpg', got %s", key)
	}
}

func TestClient_UploadObject_Error(t *testing.T) {
	uploader := &mockUploader{
		err: errors.New("upload failed"),
	}
	client := &Client{
		Client:   &s3.Client{},
		Bucket:   aws.String("test-bucket"),
		uploader: uploader,
	}

	_, err := client.UploadObject(context.Background(), "test/key.jpg", "image/jpeg", nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestClient_UploadObject_EmptyKey(t *testing.T) {
	uploader := &mockUploader{
		output: &s3transfermanager.UploadObjectOutput{
			Key: nil,
		},
	}
	client := &Client{
		Client:   &s3.Client{},
		Bucket:   aws.String("test-bucket"),
		uploader: uploader,
	}

	_, err := client.UploadObject(context.Background(), "test/key.jpg", "image/jpeg", nil)
	if err == nil {
		t.Error("expected error for empty key, got nil")
	}
}

func TestClient_GetObject(t *testing.T) {
	downloader := &mockDownloader{
		output: &s3transfermanager.GetObjectOutput{
			Body: io.NopCloser(bytes.NewReader([]byte("test content"))),
		},
	}
	client := &Client{
		Client:     &s3.Client{},
		Bucket:     aws.String("test-bucket"),
		downloader: downloader,
	}

	data, err := client.GetObject(context.Background(), "test/key.jpg")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if string(data) != "test content" {
		t.Errorf("expected 'test content', got %s", string(data))
	}
}

func TestClient_GetObject_Error(t *testing.T) {
	downloader := &mockDownloader{
		err: errors.New("download failed"),
	}
	client := &Client{
		Client:     &s3.Client{},
		Bucket:     aws.String("test-bucket"),
		downloader: downloader,
	}

	_, err := client.GetObject(context.Background(), "test/key.jpg")
	if err == nil {
		t.Error("expected error, got nil")
	}
}
