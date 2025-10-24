package sys

import (
	"fmt"
	"os/exec"
	"runtime"
)

func SetDns(device string, servers []string) error {
	switch runtime.GOOS {
	case "darwin":
		return setDnsDarwin(device, servers)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

func setDnsDarwin(device string, servers []string) error {
	cmd := []string{"-setdnsservers", device}
	cmd = append(cmd, servers...)

	if err := exec.Command("networksetup", cmd...).Run(); err != nil {
		return err
	}
	if err := exec.Command("dscacheutil", "-flushcache").Run(); err != nil {
		return err
	}
	return nil
}
