//go:build darwin

package route

import (
	"context"
	"net"
	"net/http"
	"syscall"

	"golang.org/x/sys/unix"
)

type darwinPeerContextKey struct{}

type darwinPeerInfo struct {
	UID    uint32
	GID    uint32
	HasGID bool
}

type syscallConn interface {
	SyscallConn() (syscall.RawConn, error)
}

func configurePipeServer(server *http.Server) {
	server.ConnContext = func(ctx context.Context, conn net.Conn) context.Context {
		info, ok := getDarwinPeerInfo(conn)
		if !ok {
			return ctx
		}
		return context.WithValue(ctx, darwinPeerContextKey{}, info)
	}
}

func getDarwinPeerInfo(conn net.Conn) (darwinPeerInfo, bool) {
	sysConn, ok := conn.(syscallConn)
	if !ok {
		return darwinPeerInfo{}, false
	}

	rawConn, err := sysConn.SyscallConn()
	if err != nil {
		return darwinPeerInfo{}, false
	}

	var (
		info darwinPeerInfo
		okay bool
	)
	if err := rawConn.Control(func(fd uintptr) {
		cred, sockErr := unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
		if sockErr != nil || cred == nil {
			return
		}
		info = darwinPeerInfo{UID: cred.Uid}
		if cred.Ngroups > 0 {
			info.GID = cred.Groups[0]
			info.HasGID = true
		}
		okay = true
	}); err != nil {
		return darwinPeerInfo{}, false
	}

	return info, okay
}

func getRequestDarwinPeerInfo(r *http.Request) (darwinPeerInfo, bool) {
	info, ok := r.Context().Value(darwinPeerContextKey{}).(darwinPeerInfo)
	return info, ok
}
