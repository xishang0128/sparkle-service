//go:build !windows && !linux && !darwin

package route

import "net/http"

func getRequestPrincipal(_ *http.Request) (string, string, bool, error) {
	return "", "", false, nil
}
