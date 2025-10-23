package route

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const maxTimestampDrift = 1 * time.Minute

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timestamp := r.Header.Get("X-Timestamp")
		signature := r.Header.Get("X-Signature")

		if timestamp == "" || signature == "" {
			sendError(w, fmt.Errorf("缺少认证信息"))
			return
		}

		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			sendError(w, fmt.Errorf("无效的时间戳格式"))
			return
		}

		requestTime := time.Unix(ts, 0)
		now := time.Now()
		timeDiff := now.Sub(requestTime)

		if timeDiff < -maxTimestampDrift || timeDiff > maxTimestampDrift {
			sendError(w, fmt.Errorf("请求已过期或时间戳无效"))
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
