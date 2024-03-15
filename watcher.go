package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/jorben/rsync-object-storage/config"
	"github.com/jorben/rsync-object-storage/helper"
	"github.com/jorben/rsync-object-storage/log"
	"io/fs"
	"path/filepath"
)

type Watcher struct {
	Enable      bool
	Ignore      []string
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
		// Hold住进程，以便单用check job的场景
		<-make(chan struct{})
	}

	if err := w.Add(w.LocalPrefix); err != nil {
		return err
	}

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
				w.PutChan <- event.Name
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

		case err, ok := <-w.Notify.Errors:
			if !ok {
				continue
			}
			log.Error(err.Error())
		}

	}
}
