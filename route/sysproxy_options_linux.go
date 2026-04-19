//go:build linux

package route

import (
	"net/http"

	"github.com/xishang0128/sysproxy-go/sysproxy"
)

func prepareSysproxyOptions(r *http.Request, opt *sysproxy.Options) *sysproxy.Options {
	prepared := cloneSysproxyOptions(opt)

	peer, ok := getRequestUnixPeerInfo(r)
	if !ok {
		return prepared
	}

	prepared.PeerPID = peer.PID
	prepared.PeerUID = peer.UID
	prepared.PeerGID = peer.GID
	return prepared
}
