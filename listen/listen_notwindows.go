//go:build !windows

package listen

import (
	"net"
	"os"
)

func ListenNamedPipe(path string, sddl string) (net.Listener, error) {
	_, _ = path, sddl
	return nil, os.ErrInvalid
}
