//go:build linux

package coreapi

import (
	"net/http"
	corepkg "sparkle-service/core"
	"sparkle-service/route/pipectx"
)

func coreLaunchOptions(r *http.Request) []corepkg.LaunchOption {
	info, ok := pipectx.RequestUnixPeerInfo(r)
	if !ok {
		return nil
	}
	return []corepkg.LaunchOption{corepkg.WithLogFileGroup(info.GID)}
}
