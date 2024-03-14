package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/jorben/rsync-object-storage/config"
	"github.com/jorben/rsync-object-storage/log"
	"os"
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

	// 检查本地路径可读性
	if _, err = os.ReadDir(c.Local.Path); err != nil {
		log.Fatalf("ReadDir err: %s", err.Error())
	}

	// 检查对象存储桶是否存在
	s, err := NewStorage(c)
	if err != nil {
		log.Fatalf("NewStorage err: %s", err.Error())
	}
	ctx := context.Background()
	if err = s.BucketExists(ctx); err != nil {
		log.Fatalf("BucketExist err: %s", err.Error())
	}

	// 创建Watcher实例
	w, err := NewWatcher(c.Sync.Ignore)
	if err != nil {
		log.Fatalf("NewWatcher err: %s", err.Error())
	}
	defer w.Close()

	// 异步处理变更事件
	t := NewTransfer(c.Local.Path, s)
	go t.ModifyObject(ctx, w.ModifyCh)
	go t.DeleteObject(ctx, w.DeleteCh)

	// 异步处理定期对账任务

	// 监听本地路径
	if err = w.Watch(c.Local.Path); err != nil {
		log.Fatalf("Watch err: %s", err.Error())
	}

}
