//go:build linux

package pipectx

import (
	"context"
	"net"
	"net/http"
	"syscall"

	"golang.org/x/sys/unix"
)

type unixPeerContextKey struct{}

type UnixPeerInfo struct {
	PID int
	UID uint32
	GID uint32
}

type syscallConn interface {
	SyscallConn() (syscall.RawConn, error)
}

func ConfigureServer(server *http.Server) {
	server.ConnContext = func(ctx context.Context, conn net.Conn) context.Context {
		info, ok := getUnixPeerInfo(conn)
		if !ok {
			return ctx
		}
		return context.WithValue(ctx, unixPeerContextKey{}, info)
	}
}

func getUnixPeerInfo(conn net.Conn) (UnixPeerInfo, bool) {
	sysConn, ok := conn.(syscallConn)
	if !ok {
		return UnixPeerInfo{}, false
	}

	rawConn, err := sysConn.SyscallConn()
	if err != nil {
		return UnixPeerInfo{}, false
	}

	var (
		info UnixPeerInfo
		okay bool
	)
	if err := rawConn.Control(func(fd uintptr) {
		ucred, sockErr := unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
		if sockErr != nil {
			return
		}
		info = UnixPeerInfo{
			PID: int(ucred.Pid),
			UID: ucred.Uid,
			GID: ucred.Gid,
		}
		okay = info.PID > 0
	}); err != nil {
		return UnixPeerInfo{}, false
	}

	return info, okay
}

func RequestUnixPeerInfo(r *http.Request) (UnixPeerInfo, bool) {
	info, ok := r.Context().Value(unixPeerContextKey{}).(UnixPeerInfo)
	return info, ok && info.PID > 0
}
