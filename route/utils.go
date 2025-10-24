package route

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/render"
)

type Request struct {
	Server string `json:"server,omitempty"`
	Bypass string `json:"bypass,omitempty"`
	Url    string `json:"url,omitempty"`

	Device           string `json:"device,omitempty"`
	OnlyActiveDevice bool   `json:"only_active_device,omitempty"`

	Servers []string `json:"servers,omitempty"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func decodeRequest(r *http.Request, v any) error {
	if r.ContentLength > 0 {
		return render.DecodeJSON(r.Body, v)
	}
	return nil
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
