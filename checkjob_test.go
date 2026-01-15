package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jorben/rsync-object-storage/config"
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

// createTestConfig 创建测试配置
func createTestConfig() *config.SyncConfig {
	cfg := &config.SyncConfig{}
	cfg.Local.Path = "/data/local"
	cfg.Remote.Path = "remote"
	cfg.Sync.RealTime.Enable = true
	cfg.Sync.RealTime.HotDelay = 5
	cfg.Sync.CheckJob.Enable = true
	cfg.Sync.CheckJob.Interval = 1
	cfg.Sync.CheckJob.StartAt = "00:00:00"
	cfg.Sync.Symlink = enum.SymlinkSkip
	cfg.Sync.Ignore = []string{".git", "node_modules"}
	return cfg
}

// TestNewCheckJob 测试 CheckJob 创建
func TestNewCheckJob(t *testing.T) {
	putCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	storage := &Storage{
		Client:       mockClient,
		Bucket:       "test-bucket",
		LocalPrefix:  "/data/local",
		RemotePrefix: "remote",
	}

	cfg := createTestConfig()
	job := NewCheckJob(cfg, putCh, storage)

	assert.NotNil(t, job)
	assert.True(t, job.Enable)
	assert.Equal(t, 1, job.Interval)
	assert.Equal(t, "/data/local", job.LocalPrefix)
	assert.Contains(t, job.Ignore, ".git")
}

// TestNewCheckJob_StartAtParsing 测试 StartAt 时间解析
func TestNewCheckJob_StartAtParsing(t *testing.T) {
	putCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	storage := &Storage{
		Client: mockClient,
		Bucket: "test-bucket",
	}

	t.Run("有效的时间格式", func(t *testing.T) {
		cfg := createTestConfig()
		cfg.Sync.CheckJob.StartAt = "03:30:00"
		job := NewCheckJob(cfg, putCh, storage)

		assert.NotNil(t, job)
		// InitialDelay 应该是正数
		assert.True(t, job.InitialDelay > 0)
	})

	t.Run("无效的时间格式", func(t *testing.T) {
		cfg := createTestConfig()
		cfg.Sync.CheckJob.StartAt = "invalid"
		job := NewCheckJob(cfg, putCh, storage)

		assert.NotNil(t, job)
		// 无效格式应该回退到 00:00:00
	})
}

// TestNewCheckJob_IntervalMinimum 测试 Interval 最小值
func TestNewCheckJob_IntervalMinimum(t *testing.T) {
	putCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	storage := &Storage{
		Client: mockClient,
		Bucket: "test-bucket",
	}

	t.Run("Interval 为 0 时设为 1", func(t *testing.T) {
		cfg := createTestConfig()
		cfg.Sync.CheckJob.Interval = 0
		job := NewCheckJob(cfg, putCh, storage)

		assert.Equal(t, 1, job.Interval)
	})

	t.Run("Interval 为负数时设为 1", func(t *testing.T) {
		cfg := createTestConfig()
		cfg.Sync.CheckJob.Interval = -5
		job := NewCheckJob(cfg, putCh, storage)

		assert.Equal(t, 1, job.Interval)
	})
}

// TestCheckJob_Run_Disabled 测试禁用状态
func TestCheckJob_Run_Disabled(t *testing.T) {
	putCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	storage := &Storage{
		Client: mockClient,
		Bucket: "test-bucket",
	}

	cfg := createTestConfig()
	cfg.Sync.CheckJob.Enable = false
	job := NewCheckJob(cfg, putCh, storage)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		job.Run(ctx)
		done <- true
	}()

	// 立即取消
	cancel()

	// 应该快速退出
	select {
	case <-done:
		// 成功退出
	case <-time.After(time.Second):
		t.Fatal("CheckJob 未能响应 Context 取消")
	}
}

// TestCheckJob_Run_ContextCancel 测试 Context 取消
func TestCheckJob_Run_ContextCancel(t *testing.T) {
	putCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	storage := &Storage{
		Client: mockClient,
		Bucket: "test-bucket",
	}

	cfg := createTestConfig()
	cfg.Sync.CheckJob.Enable = true
	job := NewCheckJob(cfg, putCh, storage)
	// 设置一个很长的初始延迟
	job.InitialDelay = time.Hour

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		job.Run(ctx)
		done <- true
	}()

	// 稍等后取消
	time.Sleep(50 * time.Millisecond)
	cancel()

	// 应该快速退出
	select {
	case <-done:
		// 成功退出
	case <-time.After(time.Second):
		t.Fatal("CheckJob 未能响应 Context 取消")
	}
}

// TestCheckJob_Walk 测试 Walk 功能
func TestCheckJob_Walk(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	assert.NoError(t, err)

	putCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	// Mock StatObject 返回错误（文件不存在于远程，需要同步）
	mockClient.On("StatObject", mock.Anything, "test-bucket", mock.Anything, minio.StatObjectOptions{}).
		Return(minio.ObjectInfo{}, assert.AnError)

	storage := &Storage{
		Client:       mockClient,
		Bucket:       "test-bucket",
		LocalPrefix:  tmpDir,
		RemotePrefix: "remote",
		SymLink:      enum.SymlinkSkip,
	}

	job := &CheckJob{
		Enable:      true,
		PutChan:     putCh,
		LocalPrefix: tmpDir,
		Ignore:      []string{},
		Storage:     storage,
	}

	ctx := context.Background()
	job.Walk(ctx)

	// 检查是否有文件被发送到 PutChan
	select {
	case path := <-putCh:
		assert.Contains(t, path, "test.txt")
	case <-time.After(time.Second):
		t.Fatal("未收到预期的同步任务")
	}

	mockClient.AssertExpectations(t)
}

// TestCheckJob_Walk_IgnoreFiles 测试 Walk 忽略文件
func TestCheckJob_Walk_IgnoreFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建被忽略的目录
	gitDir := filepath.Join(tmpDir, ".git")
	err := os.Mkdir(gitDir, 0755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(gitDir, "config"), []byte("content"), 0644)
	assert.NoError(t, err)

	// 创建正常文件
	normalFile := filepath.Join(tmpDir, "normal.txt")
	err = os.WriteFile(normalFile, []byte("content"), 0644)
	assert.NoError(t, err)

	putCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	// Mock StatObject 返回错误（文件不存在于远程）
	mockClient.On("StatObject", mock.Anything, "test-bucket", mock.Anything, minio.StatObjectOptions{}).
		Return(minio.ObjectInfo{}, assert.AnError)

	storage := &Storage{
		Client:       mockClient,
		Bucket:       "test-bucket",
		LocalPrefix:  tmpDir,
		RemotePrefix: "remote",
		SymLink:      enum.SymlinkSkip,
	}

	job := &CheckJob{
		Enable:      true,
		PutChan:     putCh,
		LocalPrefix: tmpDir,
		Ignore:      []string{".git"},
		Storage:     storage,
	}

	ctx := context.Background()
	job.Walk(ctx)

	// 收集所有发送的路径
	var sentPaths []string
	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case path := <-putCh:
			sentPaths = append(sentPaths, path)
		case <-timeout:
			goto done
		}
	}
done:

	// 检查 .git 目录下的文件未被发送
	for _, path := range sentPaths {
		assert.NotContains(t, path, ".git")
	}
}

// TestCheckJob_Walk_SameFile 测试 Walk 跳过一致的文件
func TestCheckJob_Walk_SameFile(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "same.txt")
	err := os.WriteFile(testFile, []byte("content"), 0644)
	assert.NoError(t, err)

	putCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	// Mock StatObject 返回一致的 MD5
	mockClient.On("StatObject", mock.Anything, "test-bucket", mock.Anything, minio.StatObjectOptions{}).
		Return(minio.ObjectInfo{
			Key:  "remote/same.txt",
			ETag: "9a0364b9e99bb480dd25e1f0284c8555", // "content" 的 MD5
		}, nil)

	storage := &Storage{
		Client:       mockClient,
		Bucket:       "test-bucket",
		LocalPrefix:  tmpDir,
		RemotePrefix: "remote",
		SymLink:      enum.SymlinkSkip,
	}

	job := &CheckJob{
		Enable:      true,
		PutChan:     putCh,
		LocalPrefix: tmpDir,
		Ignore:      []string{},
		Storage:     storage,
	}

	ctx := context.Background()
	job.Walk(ctx)

	// 一致的文件不应该被发送到 PutChan
	select {
	case <-putCh:
		t.Fatal("一致的文件不应该被发送到同步队列")
	case <-time.After(200 * time.Millisecond):
		// 预期行为
	}

	mockClient.AssertExpectations(t)
}

// TestCheckJob_Walk_ContextCancel 测试 Walk 响应 Context 取消
func TestCheckJob_Walk_ContextCancel(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建多个文件
	for i := 0; i < 10; i++ {
		file := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		err := os.WriteFile(file, []byte("content"), 0644)
		assert.NoError(t, err)
	}

	putCh := make(chan string, 1) // 小缓冲区
	mockClient := new(mocks.MockObjectStorageClient)

	// Mock StatObject 返回错误
	mockClient.On("StatObject", mock.Anything, "test-bucket", mock.Anything, minio.StatObjectOptions{}).
		Return(minio.ObjectInfo{}, assert.AnError)

	storage := &Storage{
		Client:       mockClient,
		Bucket:       "test-bucket",
		LocalPrefix:  tmpDir,
		RemotePrefix: "remote",
		SymLink:      enum.SymlinkSkip,
	}

	job := &CheckJob{
		Enable:      true,
		PutChan:     putCh,
		LocalPrefix: tmpDir,
		Ignore:      []string{},
		Storage:     storage,
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		job.Walk(ctx)
		done <- true
	}()

	// 立即取消
	cancel()

	// 应该快速退出
	select {
	case <-done:
		// 成功退出
	case <-time.After(time.Second):
		t.Fatal("Walk 未能响应 Context 取消")
	}
}

// TestCheckJob_Walk_EmptyDirectory 测试 Walk 空目录
func TestCheckJob_Walk_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	putCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	// 空目录也会被尝试同步，需要 Mock StatObject
	mockClient.On("StatObject", mock.Anything, "test-bucket", mock.Anything, minio.StatObjectOptions{}).
		Return(minio.ObjectInfo{}, assert.AnError)

	storage := &Storage{
		Client:       mockClient,
		Bucket:       "test-bucket",
		LocalPrefix:  tmpDir,
		RemotePrefix: "remote",
		SymLink:      enum.SymlinkSkip,
	}

	job := &CheckJob{
		Enable:      true,
		PutChan:     putCh,
		LocalPrefix: tmpDir,
		Ignore:      []string{},
		Storage:     storage,
	}

	ctx := context.Background()
	job.Walk(ctx)

	// 空目录应该正常完成
	select {
	case <-putCh:
		// 可能会同步空目录本身
	case <-time.After(200 * time.Millisecond):
		// 也可能什么都不发送
	}
}

// TestCheckJob_Walk_NestedDirectories 测试 Walk 嵌套目录
func TestCheckJob_Walk_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建嵌套目录结构
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")
	err := os.MkdirAll(nestedDir, 0755)
	assert.NoError(t, err)

	// 在最深层创建文件
	deepFile := filepath.Join(nestedDir, "deep.txt")
	err = os.WriteFile(deepFile, []byte("deep content"), 0644)
	assert.NoError(t, err)

	putCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	// Mock StatObject 返回错误
	mockClient.On("StatObject", mock.Anything, "test-bucket", mock.Anything, minio.StatObjectOptions{}).
		Return(minio.ObjectInfo{}, assert.AnError)

	storage := &Storage{
		Client:       mockClient,
		Bucket:       "test-bucket",
		LocalPrefix:  tmpDir,
		RemotePrefix: "remote",
		SymLink:      enum.SymlinkSkip,
	}

	job := &CheckJob{
		Enable:      true,
		PutChan:     putCh,
		LocalPrefix: tmpDir,
		Ignore:      []string{},
		Storage:     storage,
	}

	ctx := context.Background()
	job.Walk(ctx)

	// 收集发送的路径
	var found bool
	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case path := <-putCh:
			if path == deepFile {
				found = true
			}
		case <-timeout:
			goto done
		}
	}
done:

	assert.True(t, found, "应该找到深层嵌套的文件")
	mockClient.AssertExpectations(t)
}
