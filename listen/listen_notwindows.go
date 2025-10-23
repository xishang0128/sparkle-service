//go:build !windows

package listen

import (
	"net"
	"os"
)

func ListenNamedPipe(path string) (net.Listener, error) {
	return nil, os.ErrInvalid
}
