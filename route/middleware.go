package route

import (
	"fmt"
	"net/http"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timestamp := r.Header.Get("X-Timestamp")
		signature := r.Header.Get("X-Signature")

		if timestamp == "" || signature == "" {
			sendError(w, fmt.Errorf("缺少认证信息"))
			return
		}

		km := GetKeyManager()
		if !km.IsInitialized() {
			sendError(w, fmt.Errorf("服务未初始化"))
			return
		}

		if err := km.VerifySignature(timestamp, signature); err != nil {
			sendError(w, fmt.Errorf("认证失败: %v", err))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RequireAuth(next http.Handler) http.Handler {
	return AuthMiddleware(next)
}
