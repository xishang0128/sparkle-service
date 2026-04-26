//go:build !windows && !linux && !darwin

package auth

import "net/http"

func getRequestPrincipal(_ *http.Request) (string, string, bool, error) {
	return "", "", false, nil
}
