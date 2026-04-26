//go:build linux

package sysproxyapi

import "net/http"

func runSysproxyAsRequestUser(_ *http.Request, fn func() error) error {
	return fn()
}
