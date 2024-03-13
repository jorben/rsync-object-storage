package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/jorben/rsync-object-storage/helper"
	"io/fs"
	"log"
	"path/filepath"
)

type Watcher struct {
	Client   *fsnotify.Watcher
	ModifyCh chan string
	DeleteCh chan string
}

func NewWatcher() (*Watcher, error) {
	// 监听本地变更事件
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		Client:   w,
		ModifyCh: make(chan string),
		DeleteCh: make(chan string),
	}, nil
}

// Watch 启动监听本地路径
func (w *Watcher) Watch(path string) error {
	// 遍历指定路径下的所有子目录（fsnotify不会递归监听）
	err := filepath.WalkDir(path, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			log.Printf("Watch add %s\n", name)
			return w.Client.Add(name)
		}
		return nil
	})
	if err != nil {
		return err
	}

	for {
		select {
		case event, ok := <-w.Client.Events:
			if !ok {
				continue
			}
			log.Printf("Got event: %v\n", event)
			// Rename时会产生两个事件，一次旧文件的Rename，一次新文件的Create
			// 如果Create的是目录，那么需要建立监听
			if event.Has(fsnotify.Create) {
				if isNewDir, err := helper.IsDir(event.Name); err != nil {
					log.Printf("IsDir err: %s\n", err.Error())
				} else if isNewDir {
					// 递归监听子文件夹（Create可能是由Rename而来，并不一定没有子目录）
					err = filepath.WalkDir(event.Name, func(name string, d fs.DirEntry, err error) error {
						if err != nil {
							return err
						}
						if d.IsDir() {
							log.Printf("Watch add %s\n", name)
							return w.Client.Add(name)
						}
						return nil
					})
					if err != nil {
						log.Printf("Watch add err: %s\n", err.Error())
					}
				}
				w.ModifyCh <- event.Name
			}

			// 文件发生变更
			if event.Has(fsnotify.Write) {
				w.ModifyCh <- event.Name
			}

			// 如果删除或改在监听列表中，则需要移除监听
			// 实验证明Remove的时候fsnotify会自动处理移除监听（包括子目录），而Rename的时候只会移除被rename的目录（不包括子目录）
			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				// 这里fsnotify使用数组来存储了监听列表，遍历查找在监听范围很大的时候效率低，未来可以优化成map来存储
				for _, name := range w.Client.WatchList() {
					// 如果是曾经监听的对象，则移除对该目录及子目录的监听
					//log.Printf("DEBUG e.Name: %s vs name in list: %s\n", event.Name, name)
					if event.Name == name || (len(name) > len(event.Name) && event.Name+"/" == name[0:len(event.Name)+1]) {
						if err := w.Client.Remove(name); err != nil {
							log.Printf("Watch remove err: %s\n", err.Error())
						}
						log.Printf("Watch remove %s\n", name)
					}
				}
				w.DeleteCh <- event.Name
			}

		case err, ok := <-w.Client.Errors:
			if !ok {
				continue
			}
			log.Printf("Got error: %s\n", err.Error())
		}

	}
}
