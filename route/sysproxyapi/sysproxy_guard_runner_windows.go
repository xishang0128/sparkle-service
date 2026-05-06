//go:build windows

package sysproxyapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/UruhaLushia/sysproxy-go/sysproxy"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const internetSettingsConnectionsRegistryPath = `Software\Microsoft\Windows\CurrentVersion\Internet Settings\Connections`
const internetSettingsRegistryPath = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

var (
	advapi32SysproxyGuard       = windows.NewLazySystemDLL("advapi32.dll")
	procRegOpenCurrentUserGuard = advapi32SysproxyGuard.NewProc("RegOpenCurrentUser")
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
	return waitWindowsProxySettingsChangeReady(ctx, opts, ready)
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
		return waitWindowsProxySettingsChangeReady(ctx, opts, ready)
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

func waitWindowsProxySettingsChangeReady(ctx context.Context, opts *sysproxy.Options, ready func()) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if opts != nil && opts.UseRegistry {
		if opts.Device != "" {
			return fmt.Errorf("注册表模式不支持指定网络设备")
		}
	}

	keys, err := openWindowsProxySettingsWatchKeys()
	if err != nil {
		return err
	}
	defer func() {
		for _, key := range keys {
			key.Close()
		}
	}()

	handles := make([]windows.Handle, 0, len(keys)+1)
	for _, key := range keys {
		event, err := windows.CreateEvent(nil, 0, 0, nil)
		if err != nil {
			closeWindowsWatchHandles(handles)
			return fmt.Errorf("创建代理设置变更事件失败：%w", err)
		}
		handles = append(handles, event)

		if err := notifyWindowsProxySettingsKeyChange(key, event); err != nil {
			closeWindowsWatchHandles(handles)
			return err
		}
	}

	cancelEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		closeWindowsWatchHandles(handles)
		return fmt.Errorf("创建代理设置守护取消事件失败：%w", err)
	}
	handles = append(handles, cancelEvent)
	defer closeWindowsWatchHandles(handles)

	if ready != nil {
		ready()
	}

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = windows.SetEvent(cancelEvent)
		case <-done:
		}
	}()

	index, err := windows.WaitForMultipleObjects(handles, false, windows.INFINITE)
	if err != nil {
		return fmt.Errorf("等待代理设置变更失败：%w", err)
	}

	if index == windows.WAIT_OBJECT_0+uint32(len(handles)-1) {
		return ctx.Err()
	}
	return nil
}

func openWindowsProxySettingsWatchKeys() ([]registry.Key, error) {
	paths := []string{
		internetSettingsRegistryPath,
		internetSettingsConnectionsRegistryPath,
	}

	keys := make([]registry.Key, 0, len(paths))
	for _, path := range paths {
		key, err := openWindowsCurrentUserKey(path, windows.KEY_NOTIFY)
		if err != nil {
			for _, opened := range keys {
				opened.Close()
			}
			return nil, fmt.Errorf("打开代理设置监听注册表失败：%s: %w", path, err)
		}
		keys = append(keys, key)
	}

	return keys, nil
}

func openWindowsCurrentUserKey(path string, access uint32) (registry.Key, error) {
	var currentUser registry.Key
	if ret, _, _ := procRegOpenCurrentUserGuard.Call(uintptr(access), uintptr(unsafe.Pointer(&currentUser))); ret != 0 {
		return 0, syscall.Errno(ret)
	}
	defer currentUser.Close()

	return registry.OpenKey(currentUser, path, access)
}

func notifyWindowsProxySettingsKeyChange(key registry.Key, event windows.Handle) error {
	filter := uint32(windows.REG_NOTIFY_CHANGE_LAST_SET | windows.REG_NOTIFY_CHANGE_NAME | windows.REG_NOTIFY_THREAD_AGNOSTIC)
	err := windows.RegNotifyChangeKeyValue(windows.Handle(key), false, filter, event, true)
	if err == nil {
		return nil
	}

	filter &^= windows.REG_NOTIFY_THREAD_AGNOSTIC
	if retryErr := windows.RegNotifyChangeKeyValue(windows.Handle(key), false, filter, event, true); retryErr != nil {
		return fmt.Errorf("监听代理设置注册表失败：%w", retryErr)
	}
	return nil
}

func closeWindowsWatchHandles(handles []windows.Handle) {
	for _, handle := range handles {
		if handle != 0 {
			_ = windows.CloseHandle(handle)
		}
	}
}
