package main

import (
	"context"
	"github.com/jorben/rsync-object-storage/log"
	"strings"
)

type Transfer struct {
	localBase     string
	storageClient *Storage
}

func NewTransfer(localBase string, storageClient *Storage) *Transfer {
	return &Transfer{
		localBase:     localBase,
		storageClient: storageClient,
	}
}

func (t *Transfer) ModifyObject(ctx context.Context, list <-chan string) {
	for name := range list {
		objectName := t.GetRemotePath(name)
		if err := t.storageClient.FPutObject(ctx, name, objectName); err != nil {
			log.Errorf("FPutObject err: %s, file: %s", err.Error(), name)
			continue
		}
		log.Debugf("FPutObject success, file: %s", name)
	}
}

func (t *Transfer) DeleteObject(ctx context.Context, list <-chan string) {
	for _ = range list {

	}
}

// GetRemotePath 把本地路径映射远端路径（不包含远端Prefix）
func (t *Transfer) GetRemotePath(localPath string) string {
	return strings.TrimLeft(strings.Replace(localPath, t.localBase, "", 1), "/")
}
