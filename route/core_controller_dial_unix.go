//go:build !windows

package route

import (
	"context"
	"fmt"
	"net"
)

func dialCoreController(ctx context.Context, network string, address string) (net.Conn, error) {
	if network != "unix" {
		return nil, fmt.Errorf("unix 核心控制器仅支持 unix")
	}

	var dialer net.Dialer
	return dialer.DialContext(ctx, "unix", address)
}
