//go:build !windows

package route

import "net/http"

func runSysproxyAsRequestUser(_ *http.Request, fn func() error) error {
	return fn()
}
