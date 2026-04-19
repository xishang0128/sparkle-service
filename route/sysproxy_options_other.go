//go:build !linux

package route

import (
	"net/http"

	"github.com/xishang0128/sysproxy-go/sysproxy"
)

func prepareSysproxyOptions(_ *http.Request, opt *sysproxy.Options) *sysproxy.Options {
	return cloneSysproxyOptions(opt)
}
