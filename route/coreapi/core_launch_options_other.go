//go:build !linux && !darwin

package coreapi

import (
	"net/http"
	corepkg "sparkle-service/core"
)

func coreLaunchOptions(_ *http.Request) []corepkg.LaunchOption {
	return nil
}
