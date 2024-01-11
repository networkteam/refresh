package refresh

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/r3labs/sse/v2"
	"github.com/rs/cors"
	"gopkg.in/cenkalti/backoff.v1"
)

type Manager struct {
	*Configuration
	ID            string
	Restart       chan bool
	cancelFunc    context.CancelFunc
	context       context.Context
	buildRequests chan WatchEvent

	liveReloadSSE *sse.Server
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

	r.startLiveReloadServer()

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
		log.
			WithField("path", event.Path).
			WithField("event", event.Type).
			Debugf("Build requested")
	default:
		// Channel is full -> ignore, since there's another pending build request
		log.Debug("Build request ignored")
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
		log.Debug("drainBuildRequests: Skip init")
		return
	}

	t := time.NewTimer(r.BuildDelay)
	for {
		select {
		case event = <-r.buildRequests:
			log.
				WithField("path", event.Path).
				WithField("event", event.Type).
				Debugf("drainBuildRequests: Skip event until timer expires")
		case <-t.C:
			log.Debug("drainBuildRequests: Timer expired")
			return
		}
	}
}

func (r *Manager) startLiveReloadServer() {
	if !r.LiveReload {
		return
	}

	r.liveReloadSSE = sse.New()
	r.liveReloadSSE.AutoReplay = false
	r.liveReloadSSE.CreateStream("refresh")

	// Start HTTP server with permissive CORS on a random port in the background and terminate on r.context done

	srv := httptest.NewServer(cors.New(cors.Options{
		AllowOriginFunc: func(origin string) bool { return true },
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(r.liveReloadSSE))

	log.Debugf("liveReload: Started server on %s", srv.URL)

	// Pass the SSE server URL and event type to the process via env vars
	r.CommandEnv = append(r.CommandEnv, "REFRESH_LIVE_RELOAD_SSE_URL="+srv.URL+"/?stream=refresh", "REFRESH_LIVE_RELOAD_SSE_EVENT="+refreshRestartEventName)

	go func() {
		<-r.context.Done()
		r.liveReloadSSE.Close()
		srv.Close()
		log.Debug("liveReload: Stopped server")
	}()
}

const refreshRestartEventName = "refresh-restart"

func (r *Manager) notifyLiveReloadRestart() {
	if r.liveReloadSSE == nil {
		return
	}

	if r.ReadynessURL != "" {
		err := r.waitForReadyness()
		if err != nil {
			log.WithError(err).Warn("liveReload: Readyness check failed")
			return
		}
	}

	log.Debug("liveReload: Notify restart")

	r.liveReloadSSE.Publish("refresh", &sse.Event{
		Event: []byte(refreshRestartEventName),
		Data:  []byte("The server has been restarted"),
	})
}

func (r *Manager) waitForReadyness() error {
	log.Debug("liveReload: Waiting for readyness")

	return backoff.Retry(func() error {
		// Check if r.context is done
		select {
		case <-r.context.Done():
			return backoff.Permanent(r.context.Err())
		default:
		}

		resp, err := http.Get(r.ReadynessURL)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		log.Debug("liveReload: Readyness check successful")
		return nil
	}, backoff.NewExponentialBackOff())
}
