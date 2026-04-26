//go:build windows

package pipectx

import (
	"context"
	"net"
	"net/http"

	"golang.org/x/sys/windows"
)

type pipeHandleContextKey struct{}

type pipeHandleConn interface {
	Handle() windows.Handle
}

func ConfigureServer(server *http.Server) {
	server.ConnContext = func(ctx context.Context, conn net.Conn) context.Context {
		pipeConn, ok := conn.(pipeHandleConn)
		if !ok {
			return ctx
		}
		return context.WithValue(ctx, pipeHandleContextKey{}, pipeConn.Handle())
	}
}

func RequestPipeHandle(r *http.Request) (windows.Handle, bool) {
	handle, ok := r.Context().Value(pipeHandleContextKey{}).(windows.Handle)
	return handle, ok && handle != 0
}
