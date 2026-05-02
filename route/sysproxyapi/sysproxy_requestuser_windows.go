//go:build windows

package sysproxyapi

import (
	"net/http"

	"github.com/UruhaLushia/sparkle-service/route/pipectx"

	"github.com/UruhaLushia/sysproxy-go/sysproxy"
)

func runSysproxyAsRequestUser(r *http.Request, fn func() error) error {
	handle, ok := pipectx.RequestPipeHandle(r)
	if !ok {
		return fn()
	}
	return sysproxy.RunAsNamedPipeClient(handle, fn)
}
