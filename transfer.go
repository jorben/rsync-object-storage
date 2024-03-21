package main

import (
	"context"
	"errors"
	"github.com/jorben/rsync-object-storage/config"
	"github.com/jorben/rsync-object-storage/enum"
	"github.com/jorben/rsync-object-storage/helper"
	"github.com/jorben/rsync-object-storage/kv"
	"github.com/jorben/rsync-object-storage/log"
	"io/fs"
	"path/filepath"
	"time"
)

type Transfer struct {
	LocalPrefix  string
	RemotePrefix string
	HotDelay     time.Duration
	PutChan      chan string
	DeleteChan   chan string
	Storage      *Storage
}

func NewTransfer(c *config.SyncConfig, putCh chan string, deleteCh chan string, storage *Storage) *Transfer {
	return &Transfer{
		LocalPrefix:  c.Local.Path,
		RemotePrefix: c.Remote.Path,
		HotDelay:     time.Duration(c.Sync.RealTime.HotDelay) * time.Minute,
		PutChan:      putCh,
		DeleteChan:   deleteCh,
		Storage:      storage,
	}
}

// Run 消费队列，执行Put和Delete
func (t *Transfer) Run(ctx context.Context) {
	for {
		select {
		case path := <-t.PutChan:
			// 路径是否存在（有一些临时文件，创建后可能立刻被删除了）
			if isExist, _ := helper.IsExist(path); !isExist {
				log.Debugf("Path is not exist %s", path)
				continue
			}

			// 是否是文件夹，文件夹需要递归其子文件（RENAME事件不会收到子文件的事件）
			err := filepath.WalkDir(path, func(subPath string, d fs.DirEntry, err error) error {
				// 将执行Put的记录加入到kv，供热点文件发现
				kv.Set(subPath, "", t.HotDelay)
				if err := t.Storage.FPutObject(ctx, subPath); err == nil {
					log.Infof("Sync success, path: %s", subPath)
				} else if errors.Is(err, enum.ErrSkipTransfer) {
					log.Debugf("Skipping %s", subPath)
				} else {
					log.Errorf("FPutObject err: %s, file: %s", err.Error(), subPath)
				}
				return nil
			})
			if err != nil {
				log.Errorf("WalkDir err: %s", err.Error())
			}
		case path := <-t.DeleteChan:
			// 要判断路径是否存在（有一些文件修改保存策略是先删除再创建，避免串到了Create的后面，导致误删）
			if isExist, _ := helper.IsExist(path); isExist {
				log.Debugf("Path is still exist %s", path)
				continue
			}

			// 如果是目录，则需要遍历删除
			if err := t.Storage.RemoveObjects(ctx, path); err != nil {
				continue
			}
			log.Infof("Remove success, path: %s", path)
		}

	}
}
