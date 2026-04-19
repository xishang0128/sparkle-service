//go:build windows

package route

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

func configurePipeServer(server *http.Server) {
	server.ConnContext = func(ctx context.Context, conn net.Conn) context.Context {
		pipeConn, ok := conn.(pipeHandleConn)
		if !ok {
			return ctx
		}
		return context.WithValue(ctx, pipeHandleContextKey{}, pipeConn.Handle())
	}
}

func getRequestPipeHandle(r *http.Request) (windows.Handle, bool) {
	handle, ok := r.Context().Value(pipeHandleContextKey{}).(windows.Handle)
	return handle, ok && handle != 0
}
