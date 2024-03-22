package main

import (
	"context"
	"fmt"
	"github.com/jorben/rsync-object-storage/config"
	"github.com/jorben/rsync-object-storage/helper"
	"github.com/jorben/rsync-object-storage/log"
	"io/fs"
	"path/filepath"
	"time"
)

// CheckJob 定时问题
type CheckJob struct {
	InitialDelay time.Duration
	Interval     int
	Enable       bool
	PutChan      chan string
	LocalPrefix  string
	Ignore       []string
	Storage      *Storage
}

// NewCheckJob 创建Job实例
func NewCheckJob(c *config.SyncConfig, ch chan string, storage *Storage) *CheckJob {
	// 计算首次执行时间
	now := time.Now()
	targetTime, err := time.ParseInLocation("2006-01-02 15:04:05",
		fmt.Sprintf("%d-%02d-%02d %s", now.Year(), now.Month(), now.Day(), c.Sync.CheckJob.StartAt), now.Location())
	if err != nil {
		// 格式不正确，设置为从0点开始
		log.Errorf("Parse StartAt err: %s, Reset start at 0:0:0", err.Error())
		targetTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}

	// 最高频率1小时一次
	if c.Sync.CheckJob.Interval < 1 {
		c.Sync.CheckJob.Interval = 1
	}

	// 如果时间已过去则+24小时看下一个启动时点
	if now.After(targetTime) {
		targetTime = targetTime.Add(24 * time.Hour)
	}

	return &CheckJob{
		InitialDelay: targetTime.Sub(now),
		Interval:     c.Sync.CheckJob.Interval,
		Enable:       c.Sync.CheckJob.Enable,
		Storage:      storage,
		LocalPrefix:  c.Local.Path,
		PutChan:      ch,
		Ignore:       c.Sync.Ignore,
	}
}

// Run Check job 启动入口
func (c *CheckJob) Run(ctx context.Context) {
	if !c.Enable {
		log.Debug("The check job is disabled")
		return
	}
	log.Debugf("The check job will start at %s", time.Now().Add(c.InitialDelay).Format("2006-01-02 15:04:05"))
	time.AfterFunc(c.InitialDelay, func() {
		// 执行首次校对任务
		go c.Walk(ctx)

		// 创建周期定时器
		ticker := time.NewTicker(time.Duration(c.Interval) * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			// 执行周期校对任务
			go c.Walk(ctx)
		}
	})
}

// Walk 遍历本地文件，对比与远端差异，存在差异的丢入变更队列
func (c *CheckJob) Walk(ctx context.Context) {
	log.Info("Check job begin")
	err := filepath.WalkDir(c.LocalPrefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Errorf("WalkDir err: %s, skipping %s", err.Error(), path)
			return filepath.SkipDir
		}
		// 在忽略名单的文件夹直接跳过,不进入
		if d.IsDir() && helper.IsIgnore(path, c.Ignore) {
			return filepath.SkipDir
		}
		// todo 空文件夹处理，非空文件夹只需处理其内部文件即可
		// todo 符号链接处理
		// 对比文件
		if !d.IsDir() && !helper.IsIgnore(path, c.Ignore) {
			if isSame := c.Storage.IsSame(ctx, path, ""); !isSame {
				// 文件存在差异，丢入变更队列
				c.PutChan <- path
				log.Infof("Differences found %s", path)
			}
		}
		return nil
	})
	if err != nil {
		log.Errorf("WalkDir err: %s", err.Error())
		return
	}
	log.Info("Check job ends")
}
