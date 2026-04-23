//go:build darwin

package route

import (
	"net/http"
	"strconv"
)

func getRequestPrincipal(r *http.Request) (string, string, bool, error) {
	info, ok := getRequestDarwinPeerInfo(r)
	if !ok {
		return "", "", false, nil
	}

	return "uid", strconv.FormatUint(uint64(info.UID), 10), true, nil
}
