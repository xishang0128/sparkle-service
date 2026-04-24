//go:build windows

package core

import (
	"fmt"
	"time"

	"sparkle-service/listen/namedpipe"
)

func SendStartupNotification(network string, address string, token string) error {
	if network != "pipe" {
		return fmt.Errorf("windows 启动通知仅支持 pipe")
	}

	conn, err := namedpipe.DialTimeout(address, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(token))
	return err
}
