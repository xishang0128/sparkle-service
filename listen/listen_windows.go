//go:build windows

package listen

import (
	"net"
	"os"

	"github.com/UruhaLushia/sparkle-service/listen/namedpipe"

	"golang.org/x/sys/windows"
)

// DefaultNamedPipeSDDL is the default Security Descriptor set on the namedpipe.
// It provides read/write access to all users and the local system.
const DefaultNamedPipeSDDL = "D:PAI(A;OICI;GWGR;;;BU)(A;OICI;GWGR;;;SY)"

func ListenNamedPipe(path string, sddl string) (net.Listener, error) {
	if override := os.Getenv("LISTEN_NAMEDPIPE_SDDL"); override != "" {
		sddl = override
	}
	if sddl == "" {
		sddl = DefaultNamedPipeSDDL
	}
	securityDescriptor, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return nil, err
	}
	namedpipeLC := namedpipe.ListenConfig{
		SecurityDescriptor: securityDescriptor,
		InputBufferSize:    256 * 1024,
		OutputBufferSize:   256 * 1024,
	}
	return namedpipeLC.Listen(path)
}
