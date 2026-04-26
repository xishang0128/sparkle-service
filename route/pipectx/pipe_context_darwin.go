//go:build darwin

package pipectx

import (
	"context"
	"net"
	"net/http"
	"syscall"

	"golang.org/x/sys/unix"
)

type darwinPeerContextKey struct{}

type DarwinPeerInfo struct {
	UID    uint32
	GID    uint32
	HasGID bool
}

type syscallConn interface {
	SyscallConn() (syscall.RawConn, error)
}

func ConfigureServer(server *http.Server) {
	server.ConnContext = func(ctx context.Context, conn net.Conn) context.Context {
		info, ok := getDarwinPeerInfo(conn)
		if !ok {
			return ctx
		}
		return context.WithValue(ctx, darwinPeerContextKey{}, info)
	}
}

func getDarwinPeerInfo(conn net.Conn) (DarwinPeerInfo, bool) {
	sysConn, ok := conn.(syscallConn)
	if !ok {
		return DarwinPeerInfo{}, false
	}

	rawConn, err := sysConn.SyscallConn()
	if err != nil {
		return DarwinPeerInfo{}, false
	}

	var (
		info DarwinPeerInfo
		okay bool
	)
	if err := rawConn.Control(func(fd uintptr) {
		cred, sockErr := unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
		if sockErr != nil || cred == nil {
			return
		}
		info = DarwinPeerInfo{UID: cred.Uid}
		if cred.Ngroups > 0 {
			info.GID = cred.Groups[0]
			info.HasGID = true
		}
		okay = true
	}); err != nil {
		return DarwinPeerInfo{}, false
	}

	return info, okay
}

func RequestDarwinPeerInfo(r *http.Request) (DarwinPeerInfo, bool) {
	info, ok := r.Context().Value(darwinPeerContextKey{}).(DarwinPeerInfo)
	return info, ok
}
