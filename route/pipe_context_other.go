//go:build !windows && !linux && !darwin

package route

import "net/http"

func configurePipeServer(_ *http.Server) {}
