//go:build !windows

package route

import "net/http"

func configurePipeServer(_ *http.Server) {}
