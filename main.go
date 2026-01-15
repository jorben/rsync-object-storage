package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/jorben/rsync-object-storage/config"
	"github.com/jorben/rsync-object-storage/kv"
	"github.com/jorben/rsync-object-storage/log"
)

//const VERSION = "1.0.0"

func main() {

	configPath := flag.String("c", "./config.yaml", "Path to the configuration file")
	flag.Parse()

	c, err := config.GetConfig(*configPath)
	if err != nil {
		fmt.Printf("Load config err: %s\n", err.Error())
		return
	}

	// 初始化日志
	log.InitLogger(c.Log)
	defer log.GetLogger().Sync()

	// 输出配置供检查
	fmt.Println(c.GetString())

	// 使用可取消的context实现优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel 缓冲大小优化：根据 Worker 数量调整，避免生产者阻塞
	// PutChan: 8 Workers * 32 = 256，足够处理批量文件变更
	// DeleteChan: 删除操作较少但可能批量，设置为 64
	PutChan := make(chan string, 256)
	DeleteChan := make(chan string, 64)

	// 检查本地路径可读性
	if _, err = os.ReadDir(c.Local.Path); err != nil {
		log.Fatalf("ReadDir err: %s", err.Error())
	}

	// 检查对象存储桶是否存在
	s, err := NewStorage(c)
	if err != nil {
		log.Fatalf("NewStorage err: %s", err.Error())
	}

	if err = s.BucketExists(ctx); err != nil {
		log.Fatalf("BucketExist err: %s", err.Error())
	}

	// 创建Watcher实例
	w, err := NewWatcher(c, PutChan, DeleteChan)
	if err != nil {
		log.Fatalf("NewWatcher err: %s", err.Error())
	}

	// 使用WaitGroup等待所有goroutine退出
	var wg sync.WaitGroup

	// 创建CheckJob实例
	j := NewCheckJob(c, PutChan, s)
	// 异步处理定期对账任务
	wg.Add(1)
	go func() {
		defer wg.Done()
		j.Run(ctx)
	}()

	// 异步处理变更事件
	t := NewTransfer(c, PutChan, DeleteChan, s)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			t.Run(ctx)
		}()
	}

	// 异步监听本地路径
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := w.Watch(ctx); err != nil {
			log.Errorf("Watch err: %s", err.Error())
		}
	}()

	// 监听系统信号，实现优雅退出
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	log.Info("Received shutdown signal, gracefully shutting down...")

	// 取消context，通知所有goroutine退出
	cancel()

	// 关闭Watcher
	w.Close()

	// 停止kv清理协程
	kv.Stop()

	// 关闭channel，通知Transfer退出
	close(PutChan)
	close(DeleteChan)

	// 等待所有goroutine退出
	wg.Wait()
	log.Info("All workers stopped, shutdown complete")
}
