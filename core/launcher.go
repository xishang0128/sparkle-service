package core

import "os/exec"

type coreLauncher interface {
	Command(*launchSession) (*exec.Cmd, error)
}

type directCoreLauncher struct{}

func (directCoreLauncher) Command(launch *launchSession) (*exec.Cmd, error) {
	cmd := exec.Command(launch.executablePath, launch.args...)
	cmd.Env = launch.env
	cmd.Dir = launch.workingDir
	configureCommand(cmd)
	return cmd, nil
}
