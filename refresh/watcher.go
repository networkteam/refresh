package refresh

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rjeczalik/notify"
)

type Watcher struct {
	ctx                context.Context
	Events             chan WatchEvent
	appRoot            string
	includedExtensions []string
	ignoredFolders     []string
}

type WatchEvent struct {
	Path string
	Type string
}

func NewWatcher(ctx context.Context, appRoot string, includedExtensions []string, ignoredFolders []string) *Watcher {

	return &Watcher{
		ctx:                ctx,
		Events:             make(chan WatchEvent, 1),
		appRoot:            appRoot,
		includedExtensions: includedExtensions,
		ignoredFolders:     ignoredFolders,
	}
}

func (w *Watcher) Start() error {
	appPath, err := filepath.Abs(w.appRoot)
	if err != nil {
		return fmt.Errorf("getting absolute app root path: %w", err)
	}

	c := make(chan notify.EventInfo, 1)
	err = notify.Watch(filepath.Join(w.appRoot, "..."), c, notify.All)
	if err != nil {
		return fmt.Errorf("watching app root recursively: %w", err)
	}
	go func() {
		defer notify.Stop(c)
		for {
			select {
			case evt := <-c:
				path := evt.Path()
				if w.isIgnoredFolder(appPath, path) {
					continue
				}
				if !w.isWatchedFile(path) {
					continue
				}
				w.Events <- WatchEvent{
					Path: evt.Path(),
					Type: evt.Event().String(),
				}
			case <-w.ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (w Watcher) isIgnoredFolder(appPath, path string) bool {
	for _, e := range w.ignoredFolders {
		if strings.HasPrefix(path, filepath.Join(appPath, e, "")) {
			return true
		}
	}
	return false
}

func (w Watcher) isWatchedFile(path string) bool {
	ext := filepath.Ext(path)

	for _, e := range w.includedExtensions {
		if strings.TrimSpace(e) == ext {
			return true
		}
	}

	return false
}
