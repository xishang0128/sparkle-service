//go:build windows

package core

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"sparkle-service/listen"
	"sparkle-service/listen/namedpipe"
)

const trafficMonitorPipeAddress = `\\.\pipe\Sparkle\mihomo`

func startTrafficMonitorProxy(launch *launchSession, sddl string) (func(), error) {
	if launch == nil || launch.controllerNet != "pipe" || launch.controllerAddr == "" {
		return nil, nil
	}

	listener, err := listen.ListenNamedPipe(trafficMonitorPipeAddress, sddl)
	if err != nil {
		return nil, fmt.Errorf("监听 %s 失败：%w", trafficMonitorPipeAddress, err)
	}

	proxy := newTrafficMonitorReverseProxy(launch.controllerAddr)
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/traffic" {
				http.NotFound(w, r)
				return
			}
			proxy.ServeHTTP(w, r)
		}),
	}

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("TrafficMonitor 兼容 pipe 服务异常退出: %v", err)
		}
	}()
	log.Printf("TrafficMonitor 兼容 pipe 监听地址: %s", listener.Addr().String())

	return func() {
		if err := server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("关闭 TrafficMonitor 兼容 pipe 失败: %v", err)
		}
	}, nil
}

func newTrafficMonitorReverseProxy(controllerAddr string) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		FlushInterval: -1,
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = "mihomo.local"
			req.URL.Path = "/traffic"
			req.URL.RawPath = ""
			req.URL.RawQuery = ""
			req.Host = "mihomo.local"
		},
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return namedpipe.DialContext(ctx, controllerAddr)
			},
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			http.Error(w, fmt.Sprintf("转发 TrafficMonitor 流量请求失败：%v", err), http.StatusBadGateway)
		},
	}
}
