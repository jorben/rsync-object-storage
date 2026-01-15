package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/jorben/rsync-object-storage/config"
	"github.com/jorben/rsync-object-storage/enum"
	"github.com/jorben/rsync-object-storage/helper"
	"github.com/jorben/rsync-object-storage/log"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"os"
	"strings"
	"time"
)

type Storage struct {
	Minio        *minio.Client
	Bucket       string
	LocalPrefix  string
	RemotePrefix string
	SymLink      string
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
		SymLink:      c.Sync.Symlink,
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
	// 文件 则需要对远端内容一致性比较，内容一致则不重复上传
	if isSame := s.IsSameV2(ctx, localPath, ""); isSame {
		return enum.ErrSkipTransfer
	}
	objectName := localPath
	// 判断是否符号链接
	if isLink, _ := helper.IsSymlink(localPath); isLink {
		switch s.SymLink {
		case enum.SymlinkSkip:
			log.Debugf("SymlinkSkip %s", localPath)
			return enum.ErrSkipTransfer
		case enum.SymlinkFile:
			if isDir, _ := helper.IsDir(localPath); !isDir {
				log.Debugf("SymlinkFile %s", localPath)
				break
			}
			// 如果是文件夹 则应用Addr策略
			log.Debugf("Dir fallthrough to SymlinkAddr %s", localPath)
			fallthrough
		case enum.SymlinkAddr:
			log.Debugf("SymlinkAddr %s", localPath)
			objectName += ".link"
			// 获取目标地址
			target, _ := helper.GetSymlinkTarget(localPath)
			// 将地址写入临时文件
			randomString, err := helper.RandomString(32)
			if err != nil {
				log.Errorf("RandomString err: %s", err.Error())
				randomString = "tmp_link_content"
			}
			linkContentFile, err := os.Create("./" + randomString)
			if err != nil {
				return err
			}
			defer linkContentFile.Close()
			defer os.Remove("./" + randomString)
			_, err = io.WriteString(linkContentFile, target)
			if err != nil {
				return err
			}
			localPath = "./" + randomString
		default:
			return enum.ErrSkipTransfer
		}
	}

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

// IsSameV2 判断本地文件和远端文件内容是否一致，相较于V1新增包含了符号链接、空文件夹的判断
func (s *Storage) IsSameV2(ctx context.Context, localPath, remotePath string) bool {
	var err error
	var localMd5 string
	if remotePath == "" {
		remotePath = s.GetRemotePath(localPath)
	}
	isLink, _ := helper.IsSymlink(localPath)
	// 判断本地路径是否符号链接
	if isLink {
		switch s.SymLink {
		case enum.SymlinkSkip:
			log.Debugf("SymlinkSkip %s", localPath)
			return true
		case enum.SymlinkFile:
			if isDir, _ := helper.IsDir(localPath); !isDir {
				log.Debugf("SymlinkFile %s", localPath)
				localMd5, err = helper.FileMd5(localPath)
				if err != nil {
					log.Errorf("MD5 error: %s", err.Error())
					return false
				}
				break
			}
			// 如果是文件夹 则应用Addr策略
			log.Debugf("Dir fallthrough to SymlinkAddr %s", localPath)
			fallthrough
		case enum.SymlinkAddr:
			log.Debugf("SymlinkAddr %s", localPath)
			remotePath += ".link"
			// 获取目标地址
			target, _ := helper.GetSymlinkTarget(localPath)
			// 计算md5值
			localMd5 = helper.StringMd5(target)
		default:
			return true
		}
	}

	// 非符号链接的目录
	if isDir, _ := helper.IsDir(localPath); !isLink && isDir {
		// 判断是否非空，非空直接过
		if isEmpty, _ := helper.IsDirEmpty(localPath); !isEmpty {
			log.Debugf("Skip dir, is not empty %s", localPath)
			return true
		} else {
			// 空目录用.keep文件构建
			remotePath += "/.keep"
			localMd5 = "d41d8cd98f00b204e9800998ecf8427e"
		}
	}

	objectInfo, err := s.Minio.StatObject(ctx, s.Bucket, remotePath, minio.StatObjectOptions{})
	if err != nil {
		// 多半是Key不存在
		log.Debugf("StatObject %s, path: %s", err.Error(), remotePath)
		return false
	}
	// 是否分片上传的文件，分片上传的Etag是各分片MD5值合并后的MD5，所以与文件MCD5不一致，且ETAG带有分片数量标识
	// 分片场景，通过校验文件大小和修改时间来判断是否一致
	if strings.Contains(objectInfo.ETag, "-") {
		fileInfo, err := os.Stat(localPath)
		if err != nil {
			log.Errorf("Stat file err: %s", err.Error())
			return false
		}
		log.Debugf("Compare big file: %s, Size: %d, ModifyTime:%s, Remote Size:%d, ModifyTime:%s",
			localPath, fileInfo.Size(), fileInfo.ModTime().Format("2006-01-02 15:04:05"),
			objectInfo.Size, objectInfo.LastModified.In(time.Now().Location()).Format("2006-01-02 15:04:05"))
		if fileInfo.Size() != objectInfo.Size || fileInfo.ModTime().After(objectInfo.LastModified) {
			return false
		}
		return true
	}

	// 计算本地文件的md5
	if localMd5 == "" {
		localMd5, _ = helper.FileMd5(localPath)
	}
	log.Debugf("Compare %s, Local Md5: %s, Remote ETag: %s", localPath, localMd5, objectInfo.ETag)
	if strings.ToLower(localMd5) == strings.ToLower(objectInfo.ETag) {
		return true
	}
	return false

}

// GetRemotePath 把本地路径映射远端路径
func (s *Storage) GetRemotePath(path string) string {
	return strings.TrimLeft(strings.Replace(path, s.LocalPrefix, s.RemotePrefix, 1), "/")
}
