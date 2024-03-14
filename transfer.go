package main

import (
	"context"
	"github.com/jorben/rsync-object-storage/helper"
	"github.com/jorben/rsync-object-storage/log"
	"io/fs"
	"path/filepath"
	"strings"
)

type Transfer struct {
	localBase     string
	remoteBase    string
	storageClient *Storage
}

func NewTransfer(localBase string, remoteBase string, storageClient *Storage) *Transfer {
	return &Transfer{
		localBase:     localBase,
		remoteBase:    remoteBase,
		storageClient: storageClient,
	}
}

func (t *Transfer) ModifyObject(ctx context.Context, list <-chan string) {
	for name := range list {
		// 路径是否存在（有一些临时文件，创建后可能立刻被删除了）
		if isExist, _ := helper.IsExist(name); !isExist {
			log.Debugf("Path is not exist %s", name)
			continue
		}

		// 是否是文件夹，文件夹需要递归其子文件（RENAME事件不会收到子文件的事件）
		err := filepath.WalkDir(name, func(subPath string, d fs.DirEntry, err error) error {
			objectName := t.GetRemotePath(subPath)
			if err := t.storageClient.FPutObject(ctx, subPath, objectName); err != nil {
				log.Errorf("FPutObject err: %s, file: %s", err.Error(), subPath)
				return filepath.SkipDir
			}
			log.Debugf("FPutObject success, file: %s", subPath)
			return nil
		})
		if err != nil {
			log.Errorf("WalkDir err: %s", err.Error())
		}
	}
}

func (t *Transfer) DeleteObject(ctx context.Context, list <-chan string) {
	for _ = range list {

	}
}

// GetRemotePath 把本地路径映射远端路径（不包含远端Prefix）
func (t *Transfer) GetRemotePath(localPath string) string {
	return strings.TrimLeft(strings.Replace(localPath, t.localBase, t.remoteBase, 1), "/")
}
