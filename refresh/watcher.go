package refresh

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/rjeczalik/notify"
)

type Watcher struct {
	ctx                context.Context
	Events             chan WatchEvent
	appRoot            string
	includedExtensions []string
	includedPatterns   []string
	ignoredFolders     []string
}

type WatchEvent struct {
	Path string
	Type string
}

func NewWatcher(ctx context.Context, appRoot string, includedExtensions []string, includedPatterns []string, ignoredFolders []string) *Watcher {

	return &Watcher{
		ctx:                ctx,
		Events:             make(chan WatchEvent, 1),
		appRoot:            appRoot,
		includedExtensions: includedExtensions,
		includedPatterns:   includedPatterns,
		ignoredFolders:     ignoredFolders,
	}
}

func (w *Watcher) Start() error {
	appPath, err := filepath.Abs(w.appRoot)
	if err != nil {
		return fmt.Errorf("getting absolute app root path: %w", err)
	}

	// Validate included patterns once up front, so a malformed glob surfaces
	// immediately instead of silently never matching in the event loop.
	for _, p := range w.includedPatterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, err := filepath.Match(p, ""); err != nil {
			log.Warnf("Invalid included_patterns entry %q: %v (it will never match)", p, err)
		}
	}

	c := make(chan notify.EventInfo, 100)
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
					log.Debugf("Ignoring change in %s (ignored folder)", path)
					continue
				}
				if !w.isWatchedFile(path) {
					log.Debugf("Ignoring change in %s (not watched file)", path)
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
	base := filepath.Base(path)
	ext := filepath.Ext(path)

	// Exact match on the last extension (unchanged, backwards compatible).
	for _, e := range w.includedExtensions {
		if strings.TrimSpace(e) == ext {
			return true
		}
	}

	// Glob match on the file name (e.g. ".env*" to catch a whole family of files).
	for _, p := range w.includedPatterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if matched, err := filepath.Match(p, base); err == nil && matched {
			return true
		}
	}

	return false
}
