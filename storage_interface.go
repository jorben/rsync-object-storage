package main

import (
	"context"

	"github.com/minio/minio-go/v7"
)

// ObjectStorageClient 对象存储客户端接口
// 抽象 minio.Client 的核心方法，便于单元测试时使用 Mock 替换
type ObjectStorageClient interface {
	// ListBuckets 列出所有 Bucket
	ListBuckets(ctx context.Context) ([]minio.BucketInfo, error)
	// BucketExists 判断 Bucket 是否存在
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	// StatObject 获取对象元信息
	StatObject(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
	// FPutObject 上传文件到对象存储
	FPutObject(ctx context.Context, bucketName, objectName, filePath string, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	// RemoveObject 删除单个对象
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
	// ListObjects 列出对象
	ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
	// RemoveObjects 批量删除对象
	RemoveObjects(ctx context.Context, bucketName string, objectsCh <-chan minio.ObjectInfo, opts minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError
}

// 确保 minio.Client 实现了 ObjectStorageClient 接口
var _ ObjectStorageClient = (*minio.Client)(nil)
