package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/jorben/rsync-object-storage/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage struct {
	Client *minio.Client
	Bucket string
	Prefix string
}

// NewStorage 获取对象存储客户端实例
func NewStorage(c *config.SyncConfig) (*Storage, error) {
	cli, err := minio.New(c.Remote.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.Remote.SecretId, c.Remote.SecretKey, ""),
		Region: c.Remote.Region,
		Secure: c.Remote.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	return &Storage{
		Client: cli,
		Bucket: c.Remote.Bucket,
		Prefix: c.Remote.Path,
	}, nil
}

// BucketExists 判断Bucket是否存在
func (s *Storage) BucketExists(ctx context.Context) error {
	exist, err := s.Client.BucketExists(ctx, s.Bucket)
	if err != nil {
		return err
	}
	if !exist {
		return errors.New(fmt.Sprintf("bucket %s is not exist", s.Bucket))
	}
	return nil
}

// FPutObject 上传文件
func (s *Storage) FPutObject(ctx context.Context, localPath string, objectName string) error {
	objectName = fmt.Sprintf("%s/%s", s.Prefix, objectName)
	if _, err := s.Client.FPutObject(ctx, s.Bucket, objectName, localPath, minio.PutObjectOptions{}); err != nil {
		return err
	}
	return nil
}
