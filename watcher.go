package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/jorben/rsync-object-storage/config"
	"github.com/jorben/rsync-object-storage/helper"
	"github.com/jorben/rsync-object-storage/kv"
	"github.com/jorben/rsync-object-storage/log"
	"io/fs"
	"path/filepath"
	"sync"
	"time"
)

type Watcher struct {
	Enable      bool
	Ignore      []string
	HotDelay    time.Duration
	LocalPrefix string
	Notify      *fsnotify.Watcher
	PutChan     chan string
	DeleteChan  chan string
}

func NewWatcher(c *config.SyncConfig, putCh chan string, deleteCh chan string) (*Watcher, error) {
	// 监听本地变更事件
	notify, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		Enable:      c.Sync.RealTime.Enable,
		HotDelay:    time.Duration(c.Sync.RealTime.HotDelay) * time.Minute,
		Notify:      notify,
		PutChan:     putCh,
		DeleteChan:  deleteCh,
		LocalPrefix: c.Local.Path,
		Ignore:      c.Sync.Ignore,
	}, nil
}

// Add 添加监听路径（排除已忽略的路径）
func (w *Watcher) Add(path string) error {
	// 遍历指定路径下的所有子目录（fsnotify不会递归监听）
	return filepath.WalkDir(path, func(subPath string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Errorf("WalkDir err: %s, skipping %s", err.Error(), subPath)
			return filepath.SkipDir
		}
		// 是文件夹且不在忽略列表中
		if d.IsDir() && !helper.IsIgnore(subPath, w.Ignore) {
			if err := w.Notify.Add(subPath); err != nil {
				log.Errorf("Watch add err: %s, skipping %s", err.Error(), subPath)
				return filepath.SkipDir
			}
			log.Debugf("Watch add %s", subPath)
		}
		return nil
	})
}

// Close 关闭Watcher实例
func (w *Watcher) Close() {
	if err := w.Notify.Close(); err != nil {
		log.Errorf("Watcher close err: %s", err.Error())
	}
}

// Watch 启动监听本地路径
func (w *Watcher) Watch() error {

	if !w.Enable {
		log.Debug("The real-time sync is disabled")
		// Hold进程，以便单用check job的场景
		<-make(chan struct{})
	}

	if err := w.Add(w.LocalPrefix); err != nil {
		return err
	}

	// 热点延迟集合
	delayKeys := make(map[string]string)
	var mu sync.Mutex

	ticker := time.NewTicker(w.HotDelay)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-w.Notify.Events:
			if !ok || event.Has(fsnotify.Chmod) {
				continue
			}
			if match := helper.IsIgnore(event.Name, w.Ignore); match {
				log.Debugf("Ignore %s", event.Name)
				continue
			}
			log.Debugf("Event %s %s", event.Op.String(), event.Name)
			// Rename时会产生两个事件，一次旧文件的Rename，一次新文件的Create
			// 如果Create的是目录，那么需要建立监听
			if event.Has(fsnotify.Create) {
				_ = w.Add(event.Name)
				w.PutChan <- event.Name
			}

			// 文件发生变更
			if event.Has(fsnotify.Write) {
				// 判断文件是否热点文件，热点文件进行延迟更新，以节省流量和操作次数
				if kv.Exists(event.Name) {
					// 丢入set中，合并多个同名文件的事件（热点降温）
					log.Debugf("Hot path, will be delay sync %s", event.Name)
					mu.Lock()
					delayKeys[event.Name] = ""
					mu.Unlock()
				} else {
					w.PutChan <- event.Name
				}
			}

			// 如果删除或改在监听列表中，则需要移除监听
			// 实验证明Remove的时候fsnotify会自动处理移除监听（包括子目录），而Rename的时候只会移除被rename的目录（不包括子目录）
			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				// 这里fsnotify使用数组来存储了监听列表，遍历查找在监听范围很大的时候效率低，未来可以优化成map来存储
				for _, name := range w.Notify.WatchList() {
					// 如果是曾经监听的对象，则移除对该目录及子目录的监听
					//log.Debugf("DEBUG e.Name: %s vs name in list: %s", event.Name, name)
					if event.Name == name || (len(name) > len(event.Name) && event.Name+"/" == name[0:len(event.Name)+1]) {
						if err := w.Notify.Remove(name); err != nil {
							log.Errorf("Watch remove err: %s", err.Error())
						}
						log.Debugf("Watch remove %s", name)
					}
				}
				w.DeleteChan <- event.Name
			}

		case <-ticker.C:
			// 处理降温后的热点数据key
			for key := range delayKeys {
				log.Debugf("Get delayed path %s", key)
				w.PutChan <- key
				mu.Lock()
				delete(delayKeys, key)
				mu.Unlock()
			}

		case err, ok := <-w.Notify.Errors:
			if !ok {
				continue
			}
			log.Error(err.Error())
		}

	}
}
