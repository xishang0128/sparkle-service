//go:build !windows && !linux

package route

import "net/http"

func configurePipeServer(_ *http.Server) {}
