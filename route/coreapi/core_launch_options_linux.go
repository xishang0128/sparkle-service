//go:build linux

package coreapi

import (
	corepkg "github.com/UruhaLushia/sparkle-service/core"
	"github.com/UruhaLushia/sparkle-service/route/pipectx"
	"net/http"
)

func coreLaunchOptions(r *http.Request) []corepkg.LaunchOption {
	info, ok := pipectx.RequestUnixPeerInfo(r)
	if !ok {
		return nil
	}
	return []corepkg.LaunchOption{corepkg.WithLogFileGroup(info.GID)}
}
