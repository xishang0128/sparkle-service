package service

import (
	"fmt"

	kservice "github.com/kardianos/service"
)

type Status string

const (
	StatusUnknown Status = "unknown"
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
)

type Controller struct{}

type noopProgram struct{}

func (p *noopProgram) Start(s kservice.Service) error {
	return nil
}

func (p *noopProgram) Stop(s kservice.Service) error {
	return nil
}

func New(program kservice.Interface, executablePath string) (kservice.Service, error) {
	return kservice.New(program, newConfig(executablePath))
}

func (c Controller) Status() (Status, error) {
	svc, err := newControlService()
	if err != nil {
		return StatusUnknown, err
	}

	status, err := svc.Status()
	if err != nil {
		return StatusUnknown, fmt.Errorf("查询服务状态失败：%w", err)
	}

	switch status {
	case kservice.StatusRunning:
		return StatusRunning, nil
	case kservice.StatusStopped:
		return StatusStopped, nil
	default:
		return StatusUnknown, nil
	}
}

func (c Controller) Stop() error {
	svc, err := newControlService()
	if err != nil {
		return err
	}
	return svc.Stop()
}

func (c Controller) Restart() error {
	svc, err := newControlService()
	if err != nil {
		return err
	}
	return svc.Restart()
}

func newControlService() (kservice.Service, error) {
	svc, err := New(&noopProgram{}, "")
	if err != nil {
		return nil, fmt.Errorf("创建服务失败：%w", err)
	}
	return svc, nil
}

func newConfig(executablePath string) *kservice.Config {
	options := make(kservice.KeyValue)
	var depends []string

	switch kservice.ChosenSystem().String() {
	case "unix-systemv":
		options["SysvScript"] = sysvScript
	case "windows-service":
	default:
		depends = append(depends, "After=network-online.target")
	}

	options["RunAtLoadOnMac"] = true

	return &kservice.Config{
		Name:         "SparkleService",
		DisplayName:  "Sparkle Service",
		Description:  "Sparkle 提权服务",
		Executable:   executablePath,
		Arguments:    []string{"service", "run"},
		Dependencies: depends,
		Option:       options,
	}
}

var sysvScript = `#!/bin/sh /etc/rc.common
DESCRIPTION="{{.Description}}"
cmd="{{.Path}}"
name="SparkleService"
pid_file="/var/run/$name.pid"

start() {
	echo "Starting $name"
	$cmd >> /dev/null 2>&1 &
	echo $! > "$pid_file"
}

stop() {
	if [ -f "$pid_file" ]; then
		kill $(cat "$pid_file") 2>/dev/null
		rm "$pid_file"
	fi
	echo "Stopped $name"
}

restart() {
	stop
	start
}

case "$1" in
	start)
		start
		;;
	stop)
		stop
		;;
	restart)
		restart
		;;
	*)
		echo "Usage: $0 {start|stop|restart}"
		exit 1
esac
exit 0
`
