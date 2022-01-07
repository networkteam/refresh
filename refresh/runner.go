package refresh

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/apex/log"
)

func (m *Manager) runner() {
	var cmd *exec.Cmd
	stopProcess := func() {
		if cmd != nil {
			// kill the previous command
			pid := cmd.Process.Pid
			log.
				WithField("pid", pid).
				Info("Stopping process")
			// TODO This does not work on windows
			cmd.Process.Signal(os.Interrupt)
			cmd.Process.Wait()
		}
	}
	for {
		select {
		case <-m.Restart:
			stopProcess()
			if m.Debug {
				bp := m.FullBuildPath()
				args := []string{"exec", bp}
				args = append(args, m.CommandFlags...)
				cmd = exec.Command("dlv", args...)
			} else {
				cmd = exec.Command(m.FullBuildPath(), m.CommandFlags...)
			}
			go func() {
				log.Info("Starting process")
				err := m.runAndListen(cmd)
				if err != nil {
					log.Error(err.Error())
				}
			}()
		case <-m.context.Done():
			stopProcess()
			return
		}
	}
}

func (m *Manager) runAndListen(cmd *exec.Cmd) error {
	cmd.Stderr = m.Stderr
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}

	cmd.Stdin = m.Stdin
	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}

	cmd.Stdout = m.Stdout
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}

	var stderr bytes.Buffer

	cmd.Stderr = io.MultiWriter(&stderr, cmd.Stderr)

	// Set the environment variables from config
	if len(m.CommandEnv) != 0 {
		cmd.Env = append(m.CommandEnv, os.Environ()...)
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
