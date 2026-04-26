//go:build windows

package sysproxyapi

import (
	"net/http"

	"sparkle-service/route/pipectx"

	"github.com/xishang0128/sysproxy-go/sysproxy"
)

func runSysproxyAsRequestUser(r *http.Request, fn func() error) error {
	handle, ok := pipectx.RequestPipeHandle(r)
	if !ok {
		return fn()
	}
	return sysproxy.RunAsNamedPipeClient(handle, fn)
}
