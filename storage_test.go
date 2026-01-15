package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jorben/rsync-object-storage/enum"
	"github.com/jorben/rsync-object-storage/log"
	"github.com/jorben/rsync-object-storage/mocks"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	// 初始化一个空的 logger 用于测试
	log.InitNopLogger()
}

// TestGetRemotePath 测试路径映射功能
func TestGetRemotePath(t *testing.T) {
	tests := []struct {
		name         string
		localPrefix  string
		remotePrefix string
		path         string
		expected     string
	}{
		{
			name:         "基本路径映射",
			localPrefix:  "/data/local",
			remotePrefix: "backup",
			path:         "/data/local/file.txt",
			expected:     "backup/file.txt",
		},
		{
			name:         "空远程前缀",
			localPrefix:  "/data/local",
			remotePrefix: "",
			path:         "/data/local/subdir/file.txt",
			expected:     "subdir/file.txt",
		},
		{
			name:         "去除前导斜杠",
			localPrefix:  "/data/local",
			remotePrefix: "/remote/path",
			path:         "/data/local/file.txt",
			expected:     "remote/path/file.txt",
		},
		{
			name:         "嵌套子目录",
			localPrefix:  "/home/user",
			remotePrefix: "cloud",
			path:         "/home/user/docs/work/report.pdf",
			expected:     "cloud/docs/work/report.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{
				LocalPrefix:  tt.localPrefix,
				RemotePrefix: tt.remotePrefix,
			}
			result := s.GetRemotePath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestListBucket 测试列出 Bucket 功能
func TestListBucket(t *testing.T) {
	ctx := context.Background()

	t.Run("成功列出 Bucket", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)
		buckets := []minio.BucketInfo{
			{Name: "bucket1", CreationDate: time.Now()},
			{Name: "bucket2", CreationDate: time.Now()},
		}
		mockClient.On("ListBuckets", ctx).Return(buckets, nil)

		s := &Storage{Client: mockClient}
		result, err := s.ListBucket(ctx)

		assert.NoError(t, err)
		assert.Equal(t, []string{"bucket1", "bucket2"}, result)
		mockClient.AssertExpectations(t)
	})

	t.Run("列出 Bucket 失败", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)
		mockClient.On("ListBuckets", ctx).Return(nil, errors.New("network error"))

		s := &Storage{Client: mockClient}
		result, err := s.ListBucket(ctx)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockClient.AssertExpectations(t)
	})
}

// TestBucketExists 测试 Bucket 是否存在
func TestBucketExists(t *testing.T) {
	ctx := context.Background()

	t.Run("Bucket 存在", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)
		mockClient.On("BucketExists", ctx, "test-bucket").Return(true, nil)

		s := &Storage{Client: mockClient, Bucket: "test-bucket"}
		err := s.BucketExists(ctx)

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Bucket 不存在", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)
		mockClient.On("BucketExists", ctx, "non-existent").Return(false, nil)

		s := &Storage{Client: mockClient, Bucket: "non-existent"}
		err := s.BucketExists(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is not exist")
		mockClient.AssertExpectations(t)
	})

	t.Run("检查失败", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)
		mockClient.On("BucketExists", ctx, "test-bucket").Return(false, errors.New("access denied"))

		s := &Storage{Client: mockClient, Bucket: "test-bucket"}
		err := s.BucketExists(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
		mockClient.AssertExpectations(t)
	})
}

// TestRemoveObject 测试删除单个对象
func TestRemoveObject(t *testing.T) {
	ctx := context.Background()

	t.Run("成功删除对象", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)
		mockClient.On("StatObject", ctx, "test-bucket", "remote/file.txt", minio.StatObjectOptions{}).
			Return(minio.ObjectInfo{Key: "remote/file.txt"}, nil)
		mockClient.On("RemoveObject", ctx, "test-bucket", "remote/file.txt", minio.RemoveObjectOptions{}).
			Return(nil)

		s := &Storage{
			Client:       mockClient,
			Bucket:       "test-bucket",
			LocalPrefix:  "/data/local",
			RemotePrefix: "remote",
		}
		err := s.RemoveObject(ctx, "/data/local/file.txt")

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("对象不存在时静默返回", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)
		mockClient.On("StatObject", ctx, "test-bucket", "remote/file.txt", minio.StatObjectOptions{}).
			Return(minio.ObjectInfo{}, errors.New("key does not exist"))

		s := &Storage{
			Client:       mockClient,
			Bucket:       "test-bucket",
			LocalPrefix:  "/data/local",
			RemotePrefix: "remote",
		}
		err := s.RemoveObject(ctx, "/data/local/file.txt")

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}

// TestIsSameV2_RegularFile 测试普通文件的一致性比较
func TestIsSameV2_RegularFile(t *testing.T) {
	ctx := context.Background()

	// 创建临时测试文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("hello world"), 0644)
	assert.NoError(t, err)

	t.Run("文件内容一致", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)
		// hello world 的 MD5
		mockClient.On("StatObject", ctx, "test-bucket", "remote/test.txt", minio.StatObjectOptions{}).
			Return(minio.ObjectInfo{
				Key:  "remote/test.txt",
				ETag: "5eb63bbbe01eeed093cb22bb8f5acdc3",
			}, nil)

		s := &Storage{
			Client:       mockClient,
			Bucket:       "test-bucket",
			LocalPrefix:  tmpDir,
			RemotePrefix: "remote",
			SymLink:      enum.SymlinkSkip,
		}
		result := s.IsSameV2(ctx, testFile, "")

		assert.True(t, result)
		mockClient.AssertExpectations(t)
	})

	t.Run("文件内容不一致", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)
		mockClient.On("StatObject", ctx, "test-bucket", "remote/test.txt", minio.StatObjectOptions{}).
			Return(minio.ObjectInfo{
				Key:  "remote/test.txt",
				ETag: "different-md5-hash",
			}, nil)

		s := &Storage{
			Client:       mockClient,
			Bucket:       "test-bucket",
			LocalPrefix:  tmpDir,
			RemotePrefix: "remote",
			SymLink:      enum.SymlinkSkip,
		}
		result := s.IsSameV2(ctx, testFile, "")

		assert.False(t, result)
		mockClient.AssertExpectations(t)
	})

	t.Run("远程文件不存在", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)
		mockClient.On("StatObject", ctx, "test-bucket", "remote/test.txt", minio.StatObjectOptions{}).
			Return(minio.ObjectInfo{}, errors.New("key does not exist"))

		s := &Storage{
			Client:       mockClient,
			Bucket:       "test-bucket",
			LocalPrefix:  tmpDir,
			RemotePrefix: "remote",
			SymLink:      enum.SymlinkSkip,
		}
		result := s.IsSameV2(ctx, testFile, "")

		assert.False(t, result)
		mockClient.AssertExpectations(t)
	})
}

// TestIsSameV2_EmptyDirectory 测试空目录的一致性比较
func TestIsSameV2_EmptyDirectory(t *testing.T) {
	ctx := context.Background()

	// 创建空目录
	tmpDir := t.TempDir()
	emptyDir := filepath.Join(tmpDir, "emptydir")
	err := os.Mkdir(emptyDir, 0755)
	assert.NoError(t, err)

	t.Run("空目录存在远端 .keep 文件", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)
		// 空文件的 MD5
		mockClient.On("StatObject", ctx, "test-bucket", "remote/emptydir/.keep", minio.StatObjectOptions{}).
			Return(minio.ObjectInfo{
				Key:  "remote/emptydir/.keep",
				ETag: "d41d8cd98f00b204e9800998ecf8427e",
			}, nil)

		s := &Storage{
			Client:       mockClient,
			Bucket:       "test-bucket",
			LocalPrefix:  tmpDir,
			RemotePrefix: "remote",
			SymLink:      enum.SymlinkSkip,
		}
		result := s.IsSameV2(ctx, emptyDir, "")

		assert.True(t, result)
		mockClient.AssertExpectations(t)
	})
}

// TestIsSameV2_NonEmptyDirectory 测试非空目录跳过
func TestIsSameV2_NonEmptyDirectory(t *testing.T) {
	ctx := context.Background()

	// 创建非空目录
	tmpDir := t.TempDir()
	nonEmptyDir := filepath.Join(tmpDir, "nonemptydir")
	err := os.Mkdir(nonEmptyDir, 0755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(nonEmptyDir, "file.txt"), []byte("content"), 0644)
	assert.NoError(t, err)

	t.Run("非空目录直接返回 true", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)

		s := &Storage{
			Client:       mockClient,
			Bucket:       "test-bucket",
			LocalPrefix:  tmpDir,
			RemotePrefix: "remote",
			SymLink:      enum.SymlinkSkip,
		}
		result := s.IsSameV2(ctx, nonEmptyDir, "")

		assert.True(t, result)
		// 非空目录不应该调用任何 S3 API
		mockClient.AssertNotCalled(t, "StatObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

// TestIsSameV2_Symlink 测试符号链接处理
func TestIsSameV2_Symlink(t *testing.T) {
	ctx := context.Background()

	// 创建符号链接
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "target.txt")
	err := os.WriteFile(targetFile, []byte("target content"), 0644)
	assert.NoError(t, err)

	linkFile := filepath.Join(tmpDir, "link.txt")
	err = os.Symlink(targetFile, linkFile)
	assert.NoError(t, err)

	t.Run("SymlinkSkip 策略跳过符号链接", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)

		s := &Storage{
			Client:       mockClient,
			Bucket:       "test-bucket",
			LocalPrefix:  tmpDir,
			RemotePrefix: "remote",
			SymLink:      enum.SymlinkSkip,
		}
		result := s.IsSameV2(ctx, linkFile, "")

		assert.True(t, result)
		mockClient.AssertNotCalled(t, "StatObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

// TestIsSameV2_MultipartUpload 测试分片上传文件的比较
func TestIsSameV2_MultipartUpload(t *testing.T) {
	ctx := context.Background()

	// 创建测试文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "bigfile.bin")
	err := os.WriteFile(testFile, []byte("large file content"), 0644)
	assert.NoError(t, err)

	fileInfo, _ := os.Stat(testFile)

	t.Run("分片上传文件大小和时间一致", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)
		mockClient.On("StatObject", ctx, "test-bucket", "remote/bigfile.bin", minio.StatObjectOptions{}).
			Return(minio.ObjectInfo{
				Key:          "remote/bigfile.bin",
				ETag:         "abc123-5", // 带分片标识的 ETag
				Size:         fileInfo.Size(),
				LastModified: time.Now().Add(time.Hour), // 远端时间更新
			}, nil)

		s := &Storage{
			Client:       mockClient,
			Bucket:       "test-bucket",
			LocalPrefix:  tmpDir,
			RemotePrefix: "remote",
			SymLink:      enum.SymlinkSkip,
		}
		result := s.IsSameV2(ctx, testFile, "")

		assert.True(t, result)
		mockClient.AssertExpectations(t)
	})

	t.Run("分片上传文件大小不一致", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)
		mockClient.On("StatObject", ctx, "test-bucket", "remote/bigfile.bin", minio.StatObjectOptions{}).
			Return(minio.ObjectInfo{
				Key:          "remote/bigfile.bin",
				ETag:         "abc123-5",
				Size:         fileInfo.Size() + 100, // 大小不同
				LastModified: time.Now().Add(time.Hour),
			}, nil)

		s := &Storage{
			Client:       mockClient,
			Bucket:       "test-bucket",
			LocalPrefix:  tmpDir,
			RemotePrefix: "remote",
			SymLink:      enum.SymlinkSkip,
		}
		result := s.IsSameV2(ctx, testFile, "")

		assert.False(t, result)
		mockClient.AssertExpectations(t)
	})
}

// TestFPutObject 测试文件上传
func TestFPutObject(t *testing.T) {
	ctx := context.Background()

	// 创建测试文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "upload.txt")
	err := os.WriteFile(testFile, []byte("upload content"), 0644)
	assert.NoError(t, err)

	t.Run("文件已存在且内容一致时跳过", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)
		// "upload content" 的正确 MD5
		mockClient.On("StatObject", ctx, "test-bucket", "remote/upload.txt", minio.StatObjectOptions{}).
			Return(minio.ObjectInfo{
				Key:  "remote/upload.txt",
				ETag: "48fdd6aacff4f07f4dda2524551b38df",
			}, nil)

		s := &Storage{
			Client:       mockClient,
			Bucket:       "test-bucket",
			LocalPrefix:  tmpDir,
			RemotePrefix: "remote",
			SymLink:      enum.SymlinkSkip,
		}

		err := s.FPutObject(ctx, testFile)

		assert.ErrorIs(t, err, enum.ErrSkipTransfer)
		mockClient.AssertExpectations(t)
	})

	t.Run("文件不存在于远程时上传", func(t *testing.T) {
		mockClient := new(mocks.MockObjectStorageClient)
		// 远程不存在
		mockClient.On("StatObject", ctx, "test-bucket", "remote/upload.txt", minio.StatObjectOptions{}).
			Return(minio.ObjectInfo{}, errors.New("key not found"))

		// Mock FPutObject
		mockClient.On("FPutObject", ctx, "test-bucket", "remote/upload.txt", mock.Anything, minio.PutObjectOptions{}).
			Return(minio.UploadInfo{}, nil)

		s := &Storage{
			Client:       mockClient,
			Bucket:       "test-bucket",
			LocalPrefix:  tmpDir,
			RemotePrefix: "remote",
			SymLink:      enum.SymlinkSkip,
		}

		err := s.FPutObject(ctx, testFile)

		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}
