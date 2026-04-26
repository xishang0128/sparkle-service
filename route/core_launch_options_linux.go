//go:build linux

package route

import (
	"net/http"
	"sparkle-service/core"
)

func coreLaunchOptions(r *http.Request) []core.LaunchOption {
	info, ok := getRequestUnixPeerInfo(r)
	if !ok {
		return nil
	}
	return []core.LaunchOption{core.WithLogFileGroup(info.GID)}
}
