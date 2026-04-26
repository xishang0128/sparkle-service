//go:build !windows && !linux && !darwin

package pipectx

import "net/http"

func ConfigureServer(_ *http.Server) {}
