package refresh

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/apex/log"
)

func (r *Manager) runner() {
	var cmd *exec.Cmd
	stopProcess := func() {
		if cmd != nil {
			// kill the previous command
			pid := cmd.Process.Pid
			log.
				WithField("pid", pid).
				Info("Stopping process")
			_ = cmd.Process.Signal(syscall.SIGTERM)
			_, _ = cmd.Process.Wait()
		}
	}
	for {
		select {
		case <-r.Restart:
			stopProcess()
			if r.Debug {
				bp := r.FullBuildPath()
				args := []string{"exec", bp}
				args = append(args, r.CommandFlags...)
				cmd = exec.Command("dlv", args...)
			} else {
				cmd = exec.Command(r.FullBuildPath(), r.CommandFlags...)
			}
			go func() {
				log.Info("Starting process")
				err := r.runAndListen(cmd)
				if err != nil {
					log.Error(err.Error())
				}
			}()
		case <-r.context.Done():
			stopProcess()
			return
		}
	}
}

func (r *Manager) runAndListen(cmd *exec.Cmd) error {
	cmd.Stderr = r.Stderr
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}

	cmd.Stdin = r.Stdin
	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}

	cmd.Stdout = r.Stdout
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}

	var stderr bytes.Buffer

	cmd.Stderr = io.MultiWriter(&stderr, cmd.Stderr)

	// Set the environment variables from config
	if len(r.CommandEnv) != 0 {
		cmd.Env = append(r.CommandEnv, os.Environ()...)
	}

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("%s\n%s", err, stderr.String())
	}

	log.
		WithField("pid", cmd.Process.Pid).
		Debugf("Running: %s", strings.Join(cmd.Args, " "))
	err = cmd.Wait()
	if _, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("%s\n%s", err, stderr.String())
	}
	return nil
}
