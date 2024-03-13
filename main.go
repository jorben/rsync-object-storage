package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/jorben/rsync-object-storage/config"
	"log"
	"os"
)

//const VERSION = "1.0.0"

func main() {

	configPath := flag.String("c", "./config.yaml", "Path to the configuration file")
	flag.Parse()

	c, err := config.GetConfig(*configPath)
	if err != nil {
		log.Fatalf("Load config err: %s\n", err.Error())
	}

	// 输出配置供检查
	fmt.Println(c.GetString())

	// 检查本地路径可读性
	if _, err = os.ReadDir(c.Local.Path); err != nil {
		log.Fatalf("ReadDir err: %s\n", err.Error())
	}

	// 检查对象存储桶是否存在
	s, err := NewStorage(c)
	if err != nil {
		log.Fatalf("NewStorage err: %s\n", err.Error())
	}
	ctx := context.Background()
	if err = s.BucketExists(ctx); err != nil {
		log.Fatalf("BucketExist err: %s\n", err.Error())
	}

	// 创建Watcher实例
	w, err := NewWatcher(c.Sync.Ignore)
	if err != nil {
		log.Fatalf("NewWatcher err: %s\n", err.Error())
	}
	defer w.Close()

	// 异步处理变更事件
	go func() {
		for {
			select {
			case _ = <-w.ModifyCh:
			case _ = <-w.DeleteCh:
			}
		}
	}()

	// 异步处理定期对账任务

	// 监听本地路径
	if err = w.Watch(c.Local.Path); err != nil {
		log.Fatalf("Watch err: %s\n", err.Error())
	}

}
