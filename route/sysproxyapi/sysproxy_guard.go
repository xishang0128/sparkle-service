package sysproxyapi

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sync"

	"github.com/UruhaLushia/sparkle-service/log"

	"github.com/UruhaLushia/sysproxy-go/sysproxy"
)

type sysproxyGuardMode string

const (
	sysproxyGuardModeProxy sysproxyGuardMode = "proxy"
	sysproxyGuardModePAC   sysproxyGuardMode = "pac"
)

type sysproxyGuardRunner interface {
	Query(*sysproxy.Options) (*sysproxy.ProxyConfig, error)
	Apply(sysproxyGuardMode, *sysproxy.Options) error
	WaitChange(context.Context, *sysproxy.Options) error
	WaitChangeReady(context.Context, *sysproxy.Options, func()) error
	Close() error
}

type sysproxyGuardSnapshot struct {
	Mode             sysproxyGuardMode
	ProxyEnable      bool
	ProxySameForAll  bool
	ProxyServers     map[string]string
	ProxyBypass      string
	ProxyPACConflict bool
	PACEnable        bool
	PACURL           string
	PACProxyConflict bool
}

type sysproxyGuardConfig struct {
	mode      sysproxyGuardMode
	opts      *sysproxy.Options
	watchOpts *sysproxy.Options
	expected  sysproxyGuardSnapshot
	runner    sysproxyGuardRunner
}

var (
	globalSysproxyGuard = &sysproxyGuardState{}
	sysproxyMutationMu  sync.Mutex
)

type sysproxyGuardState struct {
	mu         sync.Mutex
	generation uint64
	cancel     context.CancelFunc
}

func configureSysproxyGuard(r *http.Request, enabled bool, mode sysproxyGuardMode, opts *sysproxy.Options) error {
	if !enabled {
		StopGuard()
		return nil
	}

	runner, err := captureSysproxyGuardRunner(r)
	if err != nil {
		return fmt.Errorf("初始化系统代理守护失败：%w", err)
	}

	guardOpts := cloneSysproxyOptions(opts)
	current, err := runner.Query(guardOpts)
	if err != nil {
		_ = runner.Close()
		return fmt.Errorf("读取系统代理守护目标失败：%w", err)
	}

	expected := newSysproxyGuardSnapshot(mode, current)
	fillSysproxyGuardApplyOptions(mode, guardOpts, expected)

	globalSysproxyGuard.start(&sysproxyGuardConfig{
		mode:      mode,
		opts:      guardOpts,
		watchOpts: newSysproxyGuardWatchOptions(guardOpts),
		expected:  expected,
		runner:    runner,
	})

	return nil
}

func configureSysproxyGuardBestEffort(r *http.Request, enabled bool, mode sysproxyGuardMode, opts *sysproxy.Options) {
	if err := configureSysproxyGuard(r, enabled, mode, opts); err != nil {
		log.Printf("系统代理已设置，但系统代理守护启动失败：%v", err)
		publishSysproxyGuardEvent(sysproxyEventGuardWatchFailed, mode, false, "系统代理守护启动失败，已停止", err)
	}
}

func StopGuard() {
	globalSysproxyGuard.stop()
}

func runSysproxyMutation(fn func() error) error {
	sysproxyMutationMu.Lock()
	defer sysproxyMutationMu.Unlock()
	return fn()
}

func (s *sysproxyGuardState) start(config *sysproxyGuardConfig) {
	ctx, cancel := context.WithCancel(context.Background())

	s.mu.Lock()
	s.generation++
	generation := s.generation
	oldCancel := s.cancel
	s.cancel = cancel
	s.mu.Unlock()

	if oldCancel != nil {
		oldCancel()
	}

	log.Printf("系统代理守护已启动：%s", config.mode)
	publishSysproxyGuardEvent(sysproxyEventGuardStarted, config.mode, true, "系统代理守护已启动", nil)
	go s.run(ctx, generation, config)
}

func (s *sysproxyGuardState) stop() {
	s.mu.Lock()
	s.generation++
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
		log.Println("系统代理守护已停止")
		publishSysproxyGuardEvent(sysproxyEventGuardStopped, "", false, "系统代理守护已停止", nil)
	}
}

func (s *sysproxyGuardState) active(generation uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.generation == generation
}

func (s *sysproxyGuardState) run(ctx context.Context, generation uint64, config *sysproxyGuardConfig) {
	defer config.runner.Close()

	var lastErr string
	for {
		if ctx.Err() != nil {
			return
		}

		watchCtx, cancelWatch, watchReady, watchErr := startSysproxyGuardWatch(ctx, config.runner, config.watchOpts)
		select {
		case <-ctx.Done():
			cancelWatch()
			return
		case <-watchReady:
		}

		current, err := config.runner.Query(config.opts)
		if err != nil {
			errText := err.Error()
			if errText != lastErr {
				log.Printf("系统代理守护检查失败：%v", err)
				publishSysproxyGuardEvent(sysproxyEventGuardCheckFailed, config.mode, true, "系统代理守护检查失败", err)
				lastErr = errText
			}
			if !waitSysproxyGuardNextChange(ctx, cancelWatch, watchErr) {
				return
			}
			continue
		}

		if !sysproxyGuardMatches(config.mode, config.expected, current) {
			restored := false
			err = runSysproxyMutation(func() error {
				if !s.active(generation) {
					return nil
				}
				log.Println("系统代理守护检测到代理设置被修改，正在恢复")
				publishSysproxyGuardEvent(sysproxyEventGuardChanged, config.mode, true, "系统代理守护检测到代理设置被修改", nil)
				if err := config.runner.Apply(config.mode, config.opts); err != nil {
					return err
				}
				current, err := config.runner.Query(config.opts)
				if err != nil {
					return fmt.Errorf("系统代理守护恢复后检查失败：%w", err)
				}
				if !sysproxyGuardMatches(config.mode, config.expected, current) {
					return fmt.Errorf("系统代理守护恢复后状态仍不匹配：expected=%+v current=%+v", config.expected, newSysproxyGuardSnapshot(config.mode, current))
				}
				restored = true
				return nil
			})
			if err != nil {
				errText := err.Error()
				if errText != lastErr {
					log.Printf("系统代理守护恢复失败：%v", err)
					publishSysproxyGuardEvent(sysproxyEventGuardRestoreFailed, config.mode, true, "系统代理守护恢复失败", err)
					lastErr = errText
				}
				if !waitSysproxyGuardNextChange(ctx, nil, watchErr) {
					return
				}
				continue
			}
			if restored {
				publishSysproxyGuardEvent(sysproxyEventGuardRestored, config.mode, true, "系统代理守护已恢复代理设置", nil)
			}
		}

		lastErr = ""

		select {
		case <-ctx.Done():
			cancelWatch()
			return
		case err := <-watchErr:
			if err == nil {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			errText := err.Error()
			if errText != lastErr {
				log.Printf("系统代理守护等待变更失败，将继续重试：%v", err)
				publishSysproxyGuardEvent(sysproxyEventGuardWatchFailed, config.mode, true, "系统代理守护等待变更失败，将继续重试", err)
				lastErr = errText
			}
		case <-watchCtx.Done():
			if ctx.Err() != nil {
				return
			}
		}
	}
}

func startSysproxyGuardWatch(ctx context.Context, runner sysproxyGuardRunner, opts *sysproxy.Options) (context.Context, context.CancelFunc, <-chan struct{}, <-chan error) {
	watchCtx, cancel := context.WithCancel(ctx)
	ready := make(chan struct{})
	errCh := make(chan error, 1)
	var readyOnce sync.Once

	go func() {
		err := runner.WaitChangeReady(watchCtx, opts, func() {
			readyOnce.Do(func() { close(ready) })
		})
		readyOnce.Do(func() { close(ready) })
		errCh <- err
	}()

	return watchCtx, cancel, ready, errCh
}

func waitSysproxyGuardNextChange(ctx context.Context, cancelWatch context.CancelFunc, watchErr <-chan error) bool {
	if cancelWatch != nil {
		defer cancelWatch()
	}
	select {
	case <-ctx.Done():
		return false
	case <-watchErr:
		return true
	}
}

func sysproxyGuardMatches(mode sysproxyGuardMode, expected sysproxyGuardSnapshot, current *sysproxy.ProxyConfig) bool {
	return reflect.DeepEqual(expected, newSysproxyGuardSnapshot(mode, current))
}

func newSysproxyGuardSnapshot(mode sysproxyGuardMode, config *sysproxy.ProxyConfig) sysproxyGuardSnapshot {
	snapshot := sysproxyGuardSnapshot{Mode: mode}
	if config == nil {
		return snapshot
	}

	switch mode {
	case sysproxyGuardModeProxy:
		snapshot.ProxyEnable = config.Proxy.Enable
		snapshot.ProxySameForAll = config.Proxy.SameForAll
		snapshot.ProxyServers = copyStringMap(config.Proxy.Servers)
		snapshot.ProxyBypass = config.Proxy.Bypass
		snapshot.ProxyPACConflict = config.PAC.Enable
	case sysproxyGuardModePAC:
		snapshot.PACEnable = config.PAC.Enable
		snapshot.PACURL = config.PAC.URL
		snapshot.PACProxyConflict = config.Proxy.Enable
	}

	return snapshot
}

func fillSysproxyGuardApplyOptions(mode sysproxyGuardMode, opts *sysproxy.Options, expected sysproxyGuardSnapshot) {
	switch mode {
	case sysproxyGuardModeProxy:
		opts.Proxy = firstNonEmpty(
			expected.ProxyServers["http_server"],
			expected.ProxyServers["https_server"],
			expected.ProxyServers["socks_server"],
			opts.Proxy,
		)
		opts.Bypass = expected.ProxyBypass
	case sysproxyGuardModePAC:
		opts.PACURL = expected.PACURL
	}
}

func newSysproxyGuardWatchOptions(opts *sysproxy.Options) *sysproxy.Options {
	if opts == nil {
		return &sysproxy.Options{}
	}

	return &sysproxy.Options{
		Device:           opts.Device,
		OnlyActiveDevice: opts.OnlyActiveDevice,
		UseRegistry:      opts.UseRegistry,
	}
}

func copyStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[string]string, len(src))
	for key, value := range src {
		if value == "" {
			continue
		}
		dst[key] = value
	}
	if len(dst) == 0 {
		return nil
	}
	return dst
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
