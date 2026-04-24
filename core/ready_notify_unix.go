//go:build !windows

package core

import (
	"fmt"
	"net"
	"time"
)

func SendStartupNotification(network string, address string, token string) error {
	if network != "unix" {
		return fmt.Errorf("unix 启动通知仅支持 unix")
	}

	conn, err := net.DialTimeout("unix", address, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(token))
	return err
}
