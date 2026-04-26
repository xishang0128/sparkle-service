//go:build windows

package coreapi

import (
	"context"
	"fmt"
	"net"
	"sparkle-service/listen/namedpipe"
)

func dialCoreController(ctx context.Context, network string, address string) (net.Conn, error) {
	if network != "pipe" {
		return nil, fmt.Errorf("windows 核心控制器仅支持 pipe")
	}
	return namedpipe.DialContext(ctx, address)
}
