//go:build darwin

package sysproxyapi

import (
	"context"
	"net/http"

	"github.com/xishang0128/sysproxy-go/sysproxy"
)

type darwinSysproxyGuardRunner struct{}

func captureSysproxyGuardRunner(_ *http.Request) (sysproxyGuardRunner, error) {
	return darwinSysproxyGuardRunner{}, nil
}

func (darwinSysproxyGuardRunner) Query(opts *sysproxy.Options) (*sysproxy.ProxyConfig, error) {
	return querySysproxyGuardSettings(opts)
}

func (darwinSysproxyGuardRunner) Apply(mode sysproxyGuardMode, opts *sysproxy.Options) error {
	return applySysproxyGuardSettings(mode, opts)
}

func (darwinSysproxyGuardRunner) WaitChange(ctx context.Context, opts *sysproxy.Options) error {
	return sysproxy.WaitProxySettingsChange(ctx, opts)
}

func (darwinSysproxyGuardRunner) Close() error {
	return nil
}
