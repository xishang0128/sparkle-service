//go:build linux

package sysproxyapi

import (
	"net/http"

	"github.com/UruhaLushia/sparkle-service/route/pipectx"

	"github.com/UruhaLushia/sysproxy-go/sysproxy"
)

func prepareSysproxyOptions(r *http.Request, opt *sysproxy.Options) *sysproxy.Options {
	prepared := cloneSysproxyOptions(opt)

	peer, ok := pipectx.RequestUnixPeerInfo(r)
	if !ok {
		return prepared
	}

	prepared.PeerPID = peer.PID
	prepared.PeerUID = peer.UID
	prepared.PeerGID = peer.GID
	return prepared
}
