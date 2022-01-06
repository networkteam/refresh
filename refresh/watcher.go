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
	c := make(chan notify.EventInfo, 1)
	err := notify.Watch(filepath.Join(w.appRoot, "..."), c, notify.All)
	if err != nil {
		return fmt.Errorf("watching app root recursively: %w", err)
	}
	go func() {
		defer notify.Stop(c)
		for {
			select {
			case evt := <-c:
				path := evt.Path()
				if w.isIgnoredFolder(path) {
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

func (w Watcher) isIgnoredFolder(path string) bool {
	// TODO Path is probably wrongly used since it is absolute and ignored folders are relative, ignored folders need to be processed absolute to app root!

	paths := strings.Split(path, "/")
	if len(paths) <= 0 {
		return false
	}

	for _, e := range w.ignoredFolders {
		if strings.TrimSpace(e) == paths[0] {
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
