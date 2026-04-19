//go:build windows

package route

import (
	"net/http"

	"github.com/xishang0128/sysproxy-go/sysproxy"
)

func runSysproxyAsRequestUser(r *http.Request, fn func() error) error {
	handle, ok := getRequestPipeHandle(r)
	if !ok {
		return fn()
	}
	return sysproxy.RunAsNamedPipeClient(handle, fn)
}
