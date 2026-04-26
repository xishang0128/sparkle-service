//go:build darwin

package route

import (
	"net/http"
	"sparkle-service/core"
)

func coreLaunchOptions(r *http.Request) []core.LaunchOption {
	info, ok := getRequestDarwinPeerInfo(r)
	if !ok || !info.HasGID {
		return nil
	}
	return []core.LaunchOption{core.WithLogFileGroup(info.GID)}
}
