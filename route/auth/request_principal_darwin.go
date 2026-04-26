//go:build darwin

package auth

import (
	"net/http"
	"strconv"

	"sparkle-service/route/pipectx"
)

func getRequestPrincipal(r *http.Request) (string, string, bool, error) {
	info, ok := pipectx.RequestDarwinPeerInfo(r)
	if !ok {
		return "", "", false, nil
	}

	return "uid", strconv.FormatUint(uint64(info.UID), 10), true, nil
}
