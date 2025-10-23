package route

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func sendJSON(w http.ResponseWriter, status string, message string) {
	w.Header().Set("Content-Type", "application/json")
	resp := Response{
		Status:  status,
		Message: message,
	}
	json.NewEncoder(w).Encode(resp)
}

func sendError(w http.ResponseWriter, err error) {
	sendJSON(w, "error", err.Error())
}
