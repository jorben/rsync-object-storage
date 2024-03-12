package main

import (
	"context"
	"fmt"
	"github.com/jorben/rsync-object-storage/config"
	"log"
	"os"
)

//const VERSION = "1.0.0"

func main() {

	ctx := context.Background()
	c, err := config.GetConfig()
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
	if err = s.BucketExists(ctx); err != nil {
		log.Fatalf("BucketExist err: %s\n", err.Error())
	}

	// 监听本地变更事件

	// 定期对账

}
