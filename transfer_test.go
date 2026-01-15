package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jorben/rsync-object-storage/enum"
	"github.com/jorben/rsync-object-storage/kv"
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

// TestNewTransfer 测试 Transfer 创建
func TestNewTransfer(t *testing.T) {
	putCh := make(chan string, 10)
	deleteCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	storage := &Storage{
		Client:       mockClient,
		Bucket:       "test-bucket",
		LocalPrefix:  "/data/local",
		RemotePrefix: "remote",
	}

	cfg := createTestConfig()
	transfer := NewTransfer(cfg, putCh, deleteCh, storage)

	assert.NotNil(t, transfer)
	assert.Equal(t, "/data/local", transfer.LocalPrefix)
	assert.Equal(t, "remote", transfer.RemotePrefix)
	assert.Equal(t, 5*time.Minute, transfer.HotDelay)
}

// TestTransfer_Run_Put 测试 Put 操作
func TestTransfer_Run_Put(t *testing.T) {
	// 重置 KV
	kv.ResetForTest()
	defer kv.Stop()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	assert.NoError(t, err)

	putCh := make(chan string, 10)
	deleteCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	// Mock StatObject 返回错误（文件不存在于远程）
	mockClient.On("StatObject", mock.Anything, "test-bucket", mock.Anything, minio.StatObjectOptions{}).
		Return(minio.ObjectInfo{}, assert.AnError)

	// Mock FPutObject 成功
	mockClient.On("FPutObject", mock.Anything, "test-bucket", mock.Anything, mock.Anything, minio.PutObjectOptions{}).
		Return(minio.UploadInfo{}, nil)

	storage := &Storage{
		Client:       mockClient,
		Bucket:       "test-bucket",
		LocalPrefix:  tmpDir,
		RemotePrefix: "remote",
		SymLink:      enum.SymlinkSkip,
	}

	transfer := &Transfer{
		LocalPrefix:  tmpDir,
		RemotePrefix: "remote",
		HotDelay:     5 * time.Minute,
		PutChan:      putCh,
		DeleteChan:   deleteCh,
		Storage:      storage,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 启动 Transfer
	done := make(chan bool)
	go func() {
		transfer.Run(ctx)
		done <- true
	}()

	// 发送 Put 任务
	putCh <- testFile

	// 等待处理
	time.Sleep(100 * time.Millisecond)

	// 取消并等待退出
	cancel()
	close(putCh)
	close(deleteCh)
	<-done

	mockClient.AssertExpectations(t)
}

// TestTransfer_Run_Delete 测试 Delete 操作
func TestTransfer_Run_Delete(t *testing.T) {
	// 重置 KV
	kv.ResetForTest()
	defer kv.Stop()

	tmpDir := t.TempDir()
	// 注意：删除测试时文件不应该存在
	nonExistentFile := filepath.Join(tmpDir, "deleted.txt")

	putCh := make(chan string, 10)
	deleteCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	// Mock ListObjects 返回空
	listCh := make(chan minio.ObjectInfo)
	close(listCh)
	mockClient.On("ListObjects", mock.Anything, "test-bucket", mock.Anything).
		Return((<-chan minio.ObjectInfo)(listCh))

	// Mock RemoveObjects
	removeCh := make(chan minio.RemoveObjectError)
	close(removeCh)
	mockClient.On("RemoveObjects", mock.Anything, "test-bucket", mock.Anything, mock.Anything).
		Return((<-chan minio.RemoveObjectError)(removeCh))

	storage := &Storage{
		Client:       mockClient,
		Bucket:       "test-bucket",
		LocalPrefix:  tmpDir,
		RemotePrefix: "remote",
		SymLink:      enum.SymlinkSkip,
	}

	transfer := &Transfer{
		LocalPrefix:  tmpDir,
		RemotePrefix: "remote",
		HotDelay:     5 * time.Minute,
		PutChan:      putCh,
		DeleteChan:   deleteCh,
		Storage:      storage,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 启动 Transfer
	done := make(chan bool)
	go func() {
		transfer.Run(ctx)
		done <- true
	}()

	// 发送 Delete 任务
	deleteCh <- nonExistentFile

	// 等待处理
	time.Sleep(100 * time.Millisecond)

	// 取消并等待退出
	cancel()
	close(putCh)
	close(deleteCh)
	<-done

	mockClient.AssertExpectations(t)
}

// TestTransfer_Run_ContextCancel 测试 Context 取消
func TestTransfer_Run_ContextCancel(t *testing.T) {
	kv.ResetForTest()
	defer kv.Stop()

	putCh := make(chan string, 10)
	deleteCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	storage := &Storage{
		Client:       mockClient,
		Bucket:       "test-bucket",
		LocalPrefix:  "/data",
		RemotePrefix: "remote",
	}

	transfer := &Transfer{
		LocalPrefix:  "/data",
		RemotePrefix: "remote",
		HotDelay:     5 * time.Minute,
		PutChan:      putCh,
		DeleteChan:   deleteCh,
		Storage:      storage,
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		transfer.Run(ctx)
		done <- true
	}()

	// 立即取消
	cancel()

	// 应该快速退出
	select {
	case <-done:
		// 成功退出
	case <-time.After(time.Second):
		t.Fatal("Transfer 未能及时响应 Context 取消")
	}
}

// TestTransfer_Run_ChannelClose 测试 Channel 关闭
func TestTransfer_Run_ChannelClose(t *testing.T) {
	kv.ResetForTest()
	defer kv.Stop()

	putCh := make(chan string, 10)
	deleteCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	storage := &Storage{
		Client:       mockClient,
		Bucket:       "test-bucket",
		LocalPrefix:  "/data",
		RemotePrefix: "remote",
	}

	transfer := &Transfer{
		LocalPrefix:  "/data",
		RemotePrefix: "remote",
		HotDelay:     5 * time.Minute,
		PutChan:      putCh,
		DeleteChan:   deleteCh,
		Storage:      storage,
	}

	ctx := context.Background()

	done := make(chan bool)
	go func() {
		transfer.Run(ctx)
		done <- true
	}()

	// 关闭 PutChan
	close(putCh)

	// 应该快速退出
	select {
	case <-done:
		// 成功退出
	case <-time.After(time.Second):
		t.Fatal("Transfer 未能响应 Channel 关闭")
	}
}

// TestTransfer_Run_SkipNonExistent 测试跳过不存在的文件
func TestTransfer_Run_SkipNonExistent(t *testing.T) {
	kv.ResetForTest()
	defer kv.Stop()

	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.txt")

	putCh := make(chan string, 10)
	deleteCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	storage := &Storage{
		Client:       mockClient,
		Bucket:       "test-bucket",
		LocalPrefix:  tmpDir,
		RemotePrefix: "remote",
		SymLink:      enum.SymlinkSkip,
	}

	transfer := &Transfer{
		LocalPrefix:  tmpDir,
		RemotePrefix: "remote",
		HotDelay:     5 * time.Minute,
		PutChan:      putCh,
		DeleteChan:   deleteCh,
		Storage:      storage,
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		transfer.Run(ctx)
		done <- true
	}()

	// 发送不存在的文件
	putCh <- nonExistentFile

	// 等待处理
	time.Sleep(100 * time.Millisecond)

	// 取消并等待退出
	cancel()
	close(putCh)
	close(deleteCh)
	<-done

	// 不存在的文件不应该调用 FPutObject
	mockClient.AssertNotCalled(t, "FPutObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestTransfer_Run_SkipExistingPath 测试删除时跳过仍存在的路径
func TestTransfer_Run_SkipExistingPath(t *testing.T) {
	kv.ResetForTest()
	defer kv.Stop()

	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.txt")
	err := os.WriteFile(existingFile, []byte("content"), 0644)
	assert.NoError(t, err)

	putCh := make(chan string, 10)
	deleteCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	storage := &Storage{
		Client:       mockClient,
		Bucket:       "test-bucket",
		LocalPrefix:  tmpDir,
		RemotePrefix: "remote",
	}

	transfer := &Transfer{
		LocalPrefix:  tmpDir,
		RemotePrefix: "remote",
		HotDelay:     5 * time.Minute,
		PutChan:      putCh,
		DeleteChan:   deleteCh,
		Storage:      storage,
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		transfer.Run(ctx)
		done <- true
	}()

	// 发送仍存在的文件到删除队列
	deleteCh <- existingFile

	// 等待处理
	time.Sleep(100 * time.Millisecond)

	// 取消并等待退出
	cancel()
	close(putCh)
	close(deleteCh)
	<-done

	// 仍存在的路径不应该调用 RemoveObjects
	mockClient.AssertNotCalled(t, "RemoveObjects", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestTransfer_Run_DirectoryWalk 测试目录遍历
func TestTransfer_Run_DirectoryWalk(t *testing.T) {
	kv.ResetForTest()
	defer kv.Stop()

	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	assert.NoError(t, err)

	// 创建子文件
	for i := 0; i < 3; i++ {
		file := filepath.Join(subDir, "file"+string(rune('a'+i))+".txt")
		err := os.WriteFile(file, []byte("content"), 0644)
		assert.NoError(t, err)
	}

	putCh := make(chan string, 10)
	deleteCh := make(chan string, 10)
	mockClient := new(mocks.MockObjectStorageClient)

	// Mock 所有 StatObject 调用返回错误（文件不存在于远程）
	mockClient.On("StatObject", mock.Anything, "test-bucket", mock.Anything, minio.StatObjectOptions{}).
		Return(minio.ObjectInfo{}, assert.AnError)

	// Mock 所有 FPutObject 调用成功
	mockClient.On("FPutObject", mock.Anything, "test-bucket", mock.Anything, mock.Anything, minio.PutObjectOptions{}).
		Return(minio.UploadInfo{}, nil)

	storage := &Storage{
		Client:       mockClient,
		Bucket:       "test-bucket",
		LocalPrefix:  tmpDir,
		RemotePrefix: "remote",
		SymLink:      enum.SymlinkSkip,
	}

	transfer := &Transfer{
		LocalPrefix:  tmpDir,
		RemotePrefix: "remote",
		HotDelay:     5 * time.Minute,
		PutChan:      putCh,
		DeleteChan:   deleteCh,
		Storage:      storage,
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		transfer.Run(ctx)
		done <- true
	}()

	// 发送目录
	putCh <- subDir

	// 等待处理
	time.Sleep(200 * time.Millisecond)

	// 取消并等待退出
	cancel()
	close(putCh)
	close(deleteCh)
	<-done

	// 应该调用多次 FPutObject（目录 + 文件）
	mockClient.AssertExpectations(t)
}
