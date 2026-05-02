//go:build !linux && !darwin

package coreapi

import (
	corepkg "github.com/UruhaLushia/sparkle-service/core"
	"net/http"
)

func coreLaunchOptions(_ *http.Request) []corepkg.LaunchOption {
	return nil
}
