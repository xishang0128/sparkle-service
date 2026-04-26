//go:build windows || linux || darwin

package route

import (
	"fmt"

	"github.com/xishang0128/sysproxy-go/sysproxy"
)

func querySysproxyGuardSettings(opts *sysproxy.Options) (*sysproxy.ProxyConfig, error) {
	return sysproxy.QueryProxySettings(opts)
}

func applySysproxyGuardSettings(mode sysproxyGuardMode, opts *sysproxy.Options) error {
	switch mode {
	case sysproxyGuardModeProxy:
		return sysproxy.SetProxy(opts)
	case sysproxyGuardModePAC:
		return sysproxy.SetPac(opts)
	default:
		return fmt.Errorf("未知系统代理守护模式：%s", mode)
	}
}
