package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/jorben/rsync-object-storage/config"
	"github.com/jorben/rsync-object-storage/helper"
	"github.com/jorben/rsync-object-storage/log"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"os"
	"strings"
)

type Storage struct {
	Minio        *minio.Client
	Bucket       string
	LocalPrefix  string
	RemotePrefix string
}

// NewStorage 获取对象存储客户端实例
func NewStorage(c *config.SyncConfig) (*Storage, error) {
	cli, err := minio.New(c.Remote.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.Remote.SecretId, c.Remote.SecretKey, ""),
		Region: c.Remote.Region,
		Secure: c.Remote.UseSSL,
		// 可以跳过证书校验，可用于自签发证书的场景
		//Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	})
	if err != nil {
		return nil, err
	}

	return &Storage{
		Minio:        cli,
		Bucket:       c.Remote.Bucket,
		LocalPrefix:  c.Local.Path,
		RemotePrefix: c.Remote.Path,
	}, nil
}

// ListBucket 列出Bucket列表
func (s *Storage) ListBucket(ctx context.Context) ([]string, error) {
	var bucketList []string
	bucketInfoList, err := s.Minio.ListBuckets(ctx)
	if err != nil {
		return bucketList, err
	}
	for _, bucket := range bucketInfoList {
		bucketList = append(bucketList, bucket.Name)
	}
	return bucketList, nil
}

// BucketExists 判断Bucket是否存在
func (s *Storage) BucketExists(ctx context.Context) error {
	exist, err := s.Minio.BucketExists(ctx, s.Bucket)
	if err != nil {
		return err
	}
	if !exist {
		return errors.New(fmt.Sprintf("bucket %s is not exist", s.Bucket))
	}
	return nil
}

// RemoveObject 删除对象
func (s *Storage) RemoveObject(ctx context.Context, objectName string) error {
	objectName = s.GetRemotePath(objectName)
	_, err := s.Minio.StatObject(ctx, s.Bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		// 多半是Key不存在
		log.Debugf("StatObject err: %s, path: %s", err.Error(), objectName)
		return nil
	}

	return s.Minio.RemoveObject(ctx, s.Bucket, objectName, minio.RemoveObjectOptions{})

}

// RemoveObjects 批量删除对象
func (s *Storage) RemoveObjects(ctx context.Context, objectPath string) (someError error) {
	ch := make(chan minio.ObjectInfo)
	objectPath = s.GetRemotePath(objectPath)
	go func() {
		defer close(ch)
		for object := range s.Minio.ListObjects(ctx, s.Bucket,
			minio.ListObjectsOptions{Prefix: objectPath, Recursive: true}) {
			if object.Err != nil {
				log.Errorf("ListObjects err: %s", object.Err.Error())
				continue
			}
			if object.Key == objectPath ||
				(len(object.Key) > len(objectPath) && objectPath+"/" == object.Key[0:len(objectPath)+1]) {
				// 避免误删了前缀相同但非子文件，比如 abc abcd.txt
				ch <- object
				log.Infof("Will be delete %s", object.Key)
			}
		}
	}()

	someError = nil
	opts := minio.RemoveObjectsOptions{GovernanceBypass: true}
	for err := range s.Minio.RemoveObjects(ctx, s.Bucket, ch, opts) {
		someError = err.Err
		log.Errorf("RemoveObjects err: %s, path: %s", err.Err.Error(), err.ObjectName)
	}
	return someError
}

// FPutObject 上传对象
func (s *Storage) FPutObject(ctx context.Context, localPath string) error {
	objectName := localPath
	if isDir, _ := helper.IsDir(localPath); isDir {
		// 如果是文件夹则创建objectName/.keep文件，现有接口不支持直接创建空文件夹
		objectName += "/.keep"
		// 构造一个空文件用于上传
		localPath = "./.empty"
		if isExist, _ := helper.IsExist(localPath); !isExist {
			// 创建空文件
			emptyFile, err := os.Create(localPath)
			if err != nil {
				return errors.New(fmt.Sprintf("Create .keep file err: %s", err.Error()))
			}
			_ = emptyFile.Close()
		}
	} else {
		// 文件 则需要对远端内容一致性比较，内容一致则不重复上传
		if isSame := s.IsSame(ctx, localPath); isSame {
			log.Debugf("Consistent, skipping %s", localPath)
			return nil
		}
	}

	objectName = s.GetRemotePath(objectName)
	tmp := localPath
	// 先拷贝 再上传
	randomString, err := helper.RandomString(32)
	if err != nil {
		log.Errorf("RandomString err: %s", err.Error())
	} else {
		tmp = "./." + randomString
		fileSize, err := helper.Copy(localPath, tmp)
		if err == nil {
			log.Debugf("Copy is ready, size %s", helper.ByteFormat(fileSize))
			defer os.Remove(tmp)
		} else {
			log.Errorf("Copy err: %s", err.Error())
			// 拷贝失败，使用原始文件路径
			tmp = localPath
		}
	}

	if _, err := s.Minio.FPutObject(ctx, s.Bucket, objectName, tmp, minio.PutObjectOptions{}); err != nil {
		return err
	}
	return nil
}

// IsSame 判断本地文件和远端文件内容是否一致
func (s *Storage) IsSame(ctx context.Context, localPath string) bool {
	objectName := s.GetRemotePath(localPath)
	objectInfo, err := s.Minio.StatObject(ctx, s.Bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		// 多半是Key不存在
		log.Debugf("StatObject %s, path: %s", err.Error(), objectName)
		return false
	}
	// 计算本地文件的md5
	localMd5, _ := helper.Md5(localPath)
	log.Debugf("Compare file: %s, Md5: %s, Remote ETag: %s", localPath, localMd5, objectInfo.ETag)
	if localMd5 == objectInfo.ETag {
		return true
	}
	return false
}

// GetRemotePath 把本地路径映射远端路径
func (s *Storage) GetRemotePath(path string) string {
	return strings.TrimLeft(strings.Replace(path, s.LocalPrefix, s.RemotePrefix, 1), "/")
}
