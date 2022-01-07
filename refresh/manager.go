package refresh

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/apex/log"
)

type Manager struct {
	*Configuration
	ID            string
	Restart       chan bool
	cancelFunc    context.CancelFunc
	context       context.Context
	buildRequests chan WatchEvent
}

func NewWithContext(c *Configuration, ctx context.Context) *Manager {
	ctx, cancelFunc := context.WithCancel(ctx)
	m := &Manager{
		Configuration: c,
		ID:            ID(),
		Restart:       make(chan bool),
		cancelFunc:    cancelFunc,
		context:       ctx,
		// A buffered channel for build requests: there can be one scheduled build after the current build for debouncing watch changes
		buildRequests: make(chan WatchEvent, 1),
	}
	return m
}

func (r *Manager) Start() error {
	w := NewWatcher(r.context, r.AppRoot, r.IncludedExtensions, r.IgnoredFolders)
	err := w.Start()
	if err != nil {
		return err
	}

	// Select loop to process build requests sequentially
	go func() {
		for {
			select {
			case event := <-r.buildRequests:
				r.drainBuildRequests(event)
				err := r.build(event)
				if err != nil {
					log.WithError(err).Error("Build error occurred")
				}
			case <-r.context.Done():
				return
			}
		}
	}()

	// Request an initial build
	r.requestBuild(WatchEvent{Path: r.AppRoot, Type: "init"})

	if !r.Debug {
		// Select loop to process watch events from watcher and request builds
		go func() {
			for {
				select {
				case event := <-w.Events:
					r.requestBuild(event)
				case <-r.context.Done():
					return
				}
			}
		}()
	}
	r.runner()
	return nil
}

func (r *Manager) requestBuild(event WatchEvent) {
	select {
	case r.buildRequests <- event:
		// Sent event to build requests channel
	default:
		// Channel is full -> ignore, since there's another pending build request
	}
}

func (r *Manager) build(event WatchEvent) error {
	now := time.Now()
	log.
		WithField("path", event.Path).
		WithField("event", event.Type).
		Infof("Building...")

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
		Debugf("Build complete")
	r.Restart <- true
	return nil
}

// drainBuildRequests skips request build events until BuildDelay is exceeded
func (r *Manager) drainBuildRequests(event WatchEvent) {
	// Do not wait for initial build
	if event.Type == "init" {
		return
	}

	t := time.NewTimer(r.BuildDelay)
	for {
		select {
		case event = <-r.buildRequests:
		case <-t.C:
			return
		}
	}
}
