//go:build !windows

package core

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

func createNativeStartupHook(token string) (*coreStartupHook, error) {
	socketDir, err := os.MkdirTemp("", "sparkle-core-ready-*")
	if err != nil {
		return nil, fmt.Errorf("创建核心启动通知目录失败：%w", err)
	}
	if err := os.Chmod(socketDir, 0o700); err != nil {
		_ = os.RemoveAll(socketDir)
		return nil, fmt.Errorf("设置核心启动通知目录权限失败：%w", err)
	}

	socketPath := filepath.Join(socketDir, token+".sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		_ = os.RemoveAll(socketDir)
		return nil, fmt.Errorf("创建核心启动通知 UDS 失败：%w", err)
	}
	_ = os.Chmod(socketPath, 0o600)

	postUpCommand, err := startupNotifyCommand("unix", socketPath, token)
	if err != nil {
		_ = listener.Close()
		_ = os.RemoveAll(socketDir)
		return nil, err
	}

	return newCoreStartupHook(listener, token, socketPath, postUpCommand, noopShellCommand(), func() {
		_ = os.RemoveAll(socketDir)
	}), nil
}
