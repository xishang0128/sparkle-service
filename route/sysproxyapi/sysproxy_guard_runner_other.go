//go:build !windows && !linux && !darwin

package sysproxyapi

import (
	"fmt"
	"net/http"
)

func captureSysproxyGuardRunner(_ *http.Request) (sysproxyGuardRunner, error) {
	return nil, fmt.Errorf("系统代理守护不支持当前平台")
}
