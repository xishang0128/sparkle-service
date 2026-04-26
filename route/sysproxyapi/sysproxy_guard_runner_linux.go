//go:build linux

package sysproxyapi

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sparkle-service/route/pipectx"
	"strings"

	"github.com/xishang0128/sysproxy-go/sysproxy"
)

type linuxSysproxyGuardRunner struct {
	env []string
	uid uint32
	gid uint32
}

func captureSysproxyGuardRunner(r *http.Request) (sysproxyGuardRunner, error) {
	peer, ok := pipectx.RequestUnixPeerInfo(r)
	if !ok {
		return &linuxSysproxyGuardRunner{}, nil
	}

	peerEnv, err := readLinuxProcessEnv(peer.PID)
	if err != nil {
		return nil, fmt.Errorf("读取连接进程环境失败：%w", err)
	}

	return &linuxSysproxyGuardRunner{
		env: peerEnv,
		uid: peer.UID,
		gid: peer.GID,
	}, nil
}

func (r *linuxSysproxyGuardRunner) Query(opts *sysproxy.Options) (*sysproxy.ProxyConfig, error) {
	return querySysproxyGuardSettings(r.sessionOptions(opts))
}

func (r *linuxSysproxyGuardRunner) Apply(mode sysproxyGuardMode, opts *sysproxy.Options) error {
	return applySysproxyGuardSettings(mode, r.sessionOptions(opts))
}

func (r *linuxSysproxyGuardRunner) WaitChange(ctx context.Context, opts *sysproxy.Options) error {
	return sysproxy.WaitProxySettingsChange(ctx, r.sessionOptions(opts))
}

func (r *linuxSysproxyGuardRunner) Close() error {
	return nil
}

func (r *linuxSysproxyGuardRunner) sessionOptions(opts *sysproxy.Options) *sysproxy.Options {
	sessionOpts := cloneSysproxyOptions(opts)
	sessionOpts.Environment = append([]string(nil), r.env...)
	sessionOpts.PeerUID = r.uid
	sessionOpts.PeerGID = r.gid
	return sessionOpts
}

func readLinuxProcessEnv(pid int) ([]string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid))
	if err != nil {
		return nil, err
	}

	env := []string{}
	for _, item := range strings.Split(string(data), "\x00") {
		if item == "" {
			continue
		}
		if !strings.Contains(item, "=") {
			continue
		}
		env = append(env, item)
	}
	return env, nil
}
