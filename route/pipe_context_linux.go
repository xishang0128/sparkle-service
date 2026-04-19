//go:build linux

package route

import (
	"context"
	"net"
	"net/http"
	"syscall"

	"golang.org/x/sys/unix"
)

type unixPeerContextKey struct{}

type unixPeerInfo struct {
	PID int
	UID uint32
	GID uint32
}

type syscallConn interface {
	SyscallConn() (syscall.RawConn, error)
}

func configurePipeServer(server *http.Server) {
	server.ConnContext = func(ctx context.Context, conn net.Conn) context.Context {
		info, ok := getUnixPeerInfo(conn)
		if !ok {
			return ctx
		}
		return context.WithValue(ctx, unixPeerContextKey{}, info)
	}
}

func getUnixPeerInfo(conn net.Conn) (unixPeerInfo, bool) {
	sysConn, ok := conn.(syscallConn)
	if !ok {
		return unixPeerInfo{}, false
	}

	rawConn, err := sysConn.SyscallConn()
	if err != nil {
		return unixPeerInfo{}, false
	}

	var (
		info unixPeerInfo
		okay bool
	)
	if err := rawConn.Control(func(fd uintptr) {
		ucred, sockErr := unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
		if sockErr != nil {
			return
		}
		info = unixPeerInfo{
			PID: int(ucred.Pid),
			UID: ucred.Uid,
			GID: ucred.Gid,
		}
		okay = info.PID > 0
	}); err != nil {
		return unixPeerInfo{}, false
	}

	return info, okay
}

func getRequestUnixPeerInfo(r *http.Request) (unixPeerInfo, bool) {
	info, ok := r.Context().Value(unixPeerContextKey{}).(unixPeerInfo)
	return info, ok && info.PID > 0
}
