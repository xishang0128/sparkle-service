//go:build windows

package core

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func createPrivateControllerEndpoint() (string, string, func(), error) {
	token, err := randomToken(16)
	if err != nil {
		return "", "", nil, err
	}
	return "pipe", `\\.\pipe\sparkle\mihomo-core-` + token, nil, nil
}

func hardenControllerEndpoint(network string, address string) error {
	if network != "pipe" {
		return nil
	}
	if err := setRestrictedACL(address, windows.NO_INHERITANCE); err != nil {
		return fmt.Errorf("加固核心控制器 pipe 权限失败：%w", err)
	}
	return nil
}
