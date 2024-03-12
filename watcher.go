package main

import (
	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	Client *fsnotify.Watcher
	Events chan fsnotify.Event
}

func NewWatcher() (*Watcher, error) {
	// 监听本地变更事件
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	defer w.Close()

	return &Watcher{
		Client: w,
		Events: make(chan fsnotify.Event),
	}, nil
}

func (w *Watcher) Run(path string) error {
	err := w.Client.Add(path)
	if err != nil {
		return err
	}
	w.Events = w.Client.Events
	return nil
}

func (w *Watcher) EventHandler() {

}
