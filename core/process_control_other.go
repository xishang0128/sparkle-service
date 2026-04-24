//go:build !windows

package core

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

type noopProcessController struct{}

func newProcessController() processController {
	return &noopProcessController{}
}

func configureCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func setProcessPriority(pid int32, priority string) error {
	if priority == "" || priority == "PRIORITY_NORMAL" {
		return nil
	}

	nice, ok := map[string]int{
		"PRIORITY_LOW":          19,
		"PRIORITY_BELOW":        10,
		"PRIORITY_BELOW_NORMAL": 10,
		"PRIORITY_NORMAL":       0,
		"PRIORITY_ABOVE":        -5,
		"PRIORITY_ABOVE_NORMAL": -5,
		"PRIORITY_HIGH":         -10,
		"PRIORITY_HIGHEST":      -20,
	}[priority]
	if !ok {
		return fmt.Errorf("不支持的进程优先级: %s", priority)
	}

	return syscall.Setpriority(syscall.PRIO_PROCESS, int(pid), nice)
}

func (c *noopProcessController) Attach(pid int32) error {
	return nil
}

func (c *noopProcessController) PIDs() ([]int32, error) {
	return nil, nil
}

func (c *noopProcessController) Stop(pid int32) error {
	if pid <= 0 {
		return nil
	}

	_ = syscall.Kill(-int(pid), syscall.SIGTERM)
	_ = syscall.Kill(-int(pid), syscall.SIGKILL)

	proc, err := os.FindProcess(int(pid))
	if err != nil {
		return err
	}

	return proc.Kill()
}

func (c *noopProcessController) Close() error {
	return nil
}
