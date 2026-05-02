//go:build darwin

package coreapi

import (
	corepkg "github.com/UruhaLushia/sparkle-service/core"
	"github.com/UruhaLushia/sparkle-service/route/pipectx"
	"net/http"
)

func coreLaunchOptions(r *http.Request) []corepkg.LaunchOption {
	info, ok := pipectx.RequestDarwinPeerInfo(r)
	if !ok || !info.HasGID {
		return nil
	}
	return []corepkg.LaunchOption{corepkg.WithLogFileGroup(info.GID)}
}
