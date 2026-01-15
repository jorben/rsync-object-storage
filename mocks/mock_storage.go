package mocks

import (
	"context"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/mock"
)

// MockObjectStorageClient 是 ObjectStorageClient 接口的 Mock 实现
type MockObjectStorageClient struct {
	mock.Mock
}

// ListBuckets Mock 实现
func (m *MockObjectStorageClient) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]minio.BucketInfo), args.Error(1)
}

// BucketExists Mock 实现
func (m *MockObjectStorageClient) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	args := m.Called(ctx, bucketName)
	return args.Bool(0), args.Error(1)
}

// StatObject Mock 实现
func (m *MockObjectStorageClient) StatObject(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error) {
	args := m.Called(ctx, bucketName, objectName, opts)
	return args.Get(0).(minio.ObjectInfo), args.Error(1)
}

// FPutObject Mock 实现
func (m *MockObjectStorageClient) FPutObject(ctx context.Context, bucketName, objectName, filePath string, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	args := m.Called(ctx, bucketName, objectName, filePath, opts)
	return args.Get(0).(minio.UploadInfo), args.Error(1)
}

// RemoveObject Mock 实现
func (m *MockObjectStorageClient) RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error {
	args := m.Called(ctx, bucketName, objectName, opts)
	return args.Error(0)
}

// ListObjects Mock 实现
func (m *MockObjectStorageClient) ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	args := m.Called(ctx, bucketName, opts)
	if args.Get(0) == nil {
		ch := make(chan minio.ObjectInfo)
		close(ch)
		return ch
	}
	return args.Get(0).(<-chan minio.ObjectInfo)
}

// RemoveObjects Mock 实现
func (m *MockObjectStorageClient) RemoveObjects(ctx context.Context, bucketName string, objectsCh <-chan minio.ObjectInfo, opts minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError {
	args := m.Called(ctx, bucketName, objectsCh, opts)
	if args.Get(0) == nil {
		ch := make(chan minio.RemoveObjectError)
		close(ch)
		return ch
	}
	return args.Get(0).(<-chan minio.RemoveObjectError)
}
