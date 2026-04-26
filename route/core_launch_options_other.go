//go:build !linux && !darwin

package route

import (
	"net/http"
	"sparkle-service/core"
)

func coreLaunchOptions(_ *http.Request) []core.LaunchOption {
	return nil
}
