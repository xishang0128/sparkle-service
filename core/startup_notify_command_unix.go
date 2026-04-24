//go:build !windows

package core

import (
	"fmt"
	"os"
	"strings"
)

func startupNotifyCommand(network string, address string, token string) (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("读取 service 可执行文件路径失败：%w", err)
	}

	return shellQuote(executable) +
		" __core-ready --network " + shellQuote(network) +
		" --address " + shellQuote(address) +
		" --token " + shellQuote(token), nil
}

func shellQuote(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `'\''`) + `'`
}
