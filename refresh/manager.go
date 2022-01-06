package refresh

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/apex/log"
)

type Manager struct {
	*Configuration
	ID         string
	Restart    chan bool
	cancelFunc context.CancelFunc
	context    context.Context
	gil        *sync.Once
}

func New(c *Configuration) *Manager {
	return NewWithContext(c, context.Background())
}

func NewWithContext(c *Configuration, ctx context.Context) *Manager {
	ctx, cancelFunc := context.WithCancel(ctx)
	m := &Manager{
		Configuration: c,
		ID:            ID(),
		Restart:       make(chan bool),
		cancelFunc:    cancelFunc,
		context:       ctx,
		gil:           &sync.Once{},
	}
	return m
}

func (r *Manager) Start() error {
	w := NewWatcher(r.context, r.AppRoot, r.IncludedExtensions, r.IgnoredFolders)
	err := w.Start()
	if err != nil {
		return err
	}
	go r.build(WatchEvent{Path: r.AppRoot, Type: "init"})
	if !r.Debug {
		go func() {
			for {
				select {
				case event := <-w.Events:
					go r.build(event)
				case <-r.context.Done():
					break
				}
			}
		}()
	}
	r.runner()
	return nil
}

func (r *Manager) build(event WatchEvent) {
	// TODO Replace sync.Once with sending to a buffered channel to keep rebuild events
	r.gil.Do(func() {
		defer func() {
			r.gil = &sync.Once{}
		}()
		r.buildTransaction(func() error {
			now := time.Now()
			log.
				WithField("path", event.Path).
				Debugf("Rebuild on %s", event.Type)

			args := []string{"build", "-v"}
			args = append(args, r.BuildFlags...)
			args = append(args, "-o", r.FullBuildPath(), r.BuildTargetPath)
			cmd := exec.CommandContext(r.context, "go", args...)

			err := r.runAndListen(cmd)
			if err != nil {
				if strings.Contains(err.Error(), "no buildable Go source files") {
					r.cancelFunc()
					log.WithError(err).Fatal("Unable to build")
				}
				return err
			}

			tt := time.Since(now)
			log.
				WithField("pid", cmd.Process.Pid).
				WithField("duration", tt).
				Infof("Build complete")
			r.Restart <- true
			return nil
		})
	})
}

func (r *Manager) buildTransaction(fn func() error) {
	lpath := ErrorLogPath()
	err := fn()
	if err != nil {
		f, _ := os.Create(lpath)
		fmt.Fprint(f, err)
		log.WithError(err).Error("Build error occurred")
	} else {
		os.Remove(lpath)
	}
}
