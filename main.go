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

	ctx := context.Background()
	PutChan := make(chan string)
	DeleteChan := make(chan string)
	defer close(PutChan)
	defer close(DeleteChan)

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
	defer w.Close()

	// 创建CheckJob实例
	j := NewCheckJob(c, PutChan, s)
	// 异步处理定期对账任务
	go j.Run(ctx)

	// 异步处理变更事件
	t := NewTransfer(c, PutChan, DeleteChan, s)
	for i := 0; i < 8; i++ {
		go t.Run(ctx)
	}

	// 监听本地路径
	if err = w.Watch(); err != nil {
		log.Fatalf("Watch err: %s", err.Error())
	}

}
