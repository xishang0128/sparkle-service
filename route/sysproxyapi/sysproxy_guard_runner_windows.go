//go:build windows

package sysproxyapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"

	"github.com/UruhaLushia/sysproxy-go/sysproxy"
	"golang.org/x/sys/windows"
)

type windowsTokenSysproxyGuardRunner struct {
	token windows.Token
}

type windowsSysproxyGuardRunner struct{}

func captureSysproxyGuardRunner(_ *http.Request) (sysproxyGuardRunner, error) {
	var threadToken windows.Token
	err := windows.OpenThreadToken(
		windows.CurrentThread(),
		windows.TOKEN_DUPLICATE|windows.TOKEN_IMPERSONATE|windows.TOKEN_QUERY,
		true,
		&threadToken,
	)
	if err != nil {
		if errors.Is(err, windows.ERROR_NO_TOKEN) {
			return windowsSysproxyGuardRunner{}, nil
		}
		return nil, fmt.Errorf("打开线程用户令牌失败：%w", err)
	}
	defer threadToken.Close()

	var duplicated windows.Token
	if err := windows.DuplicateTokenEx(
		threadToken,
		windows.TOKEN_IMPERSONATE|windows.TOKEN_QUERY,
		nil,
		windows.SecurityImpersonation,
		windows.TokenImpersonation,
		&duplicated,
	); err != nil {
		return nil, fmt.Errorf("复制线程用户令牌失败：%w", err)
	}

	return &windowsTokenSysproxyGuardRunner{token: duplicated}, nil
}

func (windowsSysproxyGuardRunner) WaitChange(ctx context.Context, opts *sysproxy.Options) error {
	return sysproxy.WaitProxySettingsChange(ctx, opts)
}

func (windowsSysproxyGuardRunner) WaitChangeReady(ctx context.Context, opts *sysproxy.Options, ready func()) error {
	return sysproxy.WaitProxySettingsChangeReady(ctx, opts, ready)
}

func (windowsSysproxyGuardRunner) Query(opts *sysproxy.Options) (*sysproxy.ProxyConfig, error) {
	return querySysproxyGuardSettings(opts)
}

func (windowsSysproxyGuardRunner) Apply(mode sysproxyGuardMode, opts *sysproxy.Options) error {
	return applySysproxyGuardSettings(mode, opts)
}

func (windowsSysproxyGuardRunner) Close() error {
	return nil
}

func (r *windowsTokenSysproxyGuardRunner) Query(opts *sysproxy.Options) (*sysproxy.ProxyConfig, error) {
	var result *sysproxy.ProxyConfig
	err := r.run(func() error {
		var queryErr error
		result, queryErr = querySysproxyGuardSettings(opts)
		return queryErr
	})
	return result, err
}

func (r *windowsTokenSysproxyGuardRunner) Apply(mode sysproxyGuardMode, opts *sysproxy.Options) error {
	return r.run(func() error {
		return applySysproxyGuardSettings(mode, opts)
	})
}

func (r *windowsTokenSysproxyGuardRunner) WaitChange(ctx context.Context, opts *sysproxy.Options) error {
	return r.run(func() error {
		return sysproxy.WaitProxySettingsChange(ctx, opts)
	})
}

func (r *windowsTokenSysproxyGuardRunner) WaitChangeReady(ctx context.Context, opts *sysproxy.Options, ready func()) error {
	return r.run(func() error {
		return sysproxy.WaitProxySettingsChangeReady(ctx, opts, ready)
	})
}

func (r *windowsTokenSysproxyGuardRunner) run(fn func() error) (err error) {
	if r == nil || r.token == 0 {
		return fmt.Errorf("系统代理守护用户令牌无效")
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := windows.SetThreadToken(nil, r.token); err != nil {
		return fmt.Errorf("切换系统代理守护用户失败：%w", err)
	}
	defer func() {
		if revertErr := windows.RevertToSelf(); err == nil && revertErr != nil {
			err = fmt.Errorf("恢复系统代理守护用户失败：%w", revertErr)
		}
	}()

	return fn()
}

func (r *windowsTokenSysproxyGuardRunner) Close() error {
	if r == nil || r.token == 0 {
		return nil
	}

	err := r.token.Close()
	r.token = 0
	return err
}
