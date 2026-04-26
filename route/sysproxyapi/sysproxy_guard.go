package sysproxyapi

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sync"

	"sparkle-service/log"

	"github.com/xishang0128/sysproxy-go/sysproxy"
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
	mode     sysproxyGuardMode
	opts     *sysproxy.Options
	expected sysproxyGuardSnapshot
	runner   sysproxyGuardRunner
}

var globalSysproxyGuard = &sysproxyGuardState{}

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
		mode:     mode,
		opts:     guardOpts,
		expected: expected,
		runner:   runner,
	})

	return nil
}

func StopGuard() {
	globalSysproxyGuard.stop()
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
		if err := config.runner.WaitChange(ctx, config.opts); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("系统代理守护等待变更失败，已停止：%v", err)
			return
		}

		current, err := config.runner.Query(config.opts)
		if err == nil && !sysproxyGuardMatches(config.mode, config.expected, current) && s.active(generation) {
			log.Println("系统代理守护检测到代理设置被修改，正在恢复")
			err = config.runner.Apply(config.mode, config.opts)
		}

		if err != nil {
			errText := err.Error()
			if errText != lastErr {
				log.Printf("系统代理守护检查失败：%v", err)
				lastErr = errText
			}
			continue
		}
		lastErr = ""
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
