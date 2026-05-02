package coreapi

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/UruhaLushia/sparkle-service/route/httphelper"
)

func coreControllerProxy(w http.ResponseWriter, r *http.Request) {
	network, address, err := cm.ControllerEndpoint()
	if err != nil {
		httphelper.SendError(w, httphelper.ServiceUnavailable(err.Error()))
		return
	}

	targetPath := stripCoreControllerPrefix(r.URL.Path)
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = "mihomo.local"
			req.URL.Path = targetPath
			req.URL.RawPath = ""
			req.Host = "mihomo.local"
		},
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return dialCoreController(ctx, network, address)
			},
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			httphelper.SendError(w, fmt.Errorf("转发核心控制器请求失败：%w", err))
		},
	}

	proxy.ServeHTTP(w, r)
}

func stripCoreControllerPrefix(path string) string {
	for _, prefix := range []string{"/core/controller", "/controller"} {
		if after, ok := strings.CutPrefix(path, prefix); ok {
			if after == "" {
				return "/"
			}
			return after
		}
	}
	return path
}
