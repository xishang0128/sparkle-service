package route

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/render"
)

type Request struct {
	Server string `json:"server,omitempty"`
	Bypass string `json:"bypass,omitempty"`
	Url    string `json:"url,omitempty"`

	Device           string `json:"device,omitempty"`
	OnlyActiveDevice bool   `json:"only_active_device,omitempty"`
	UseRegistry      bool   `json:"use_registry,omitempty"`
	Guard            bool   `json:"guard,omitempty"`

	Servers []string `json:"servers,omitempty"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return e.Message
}

func newHTTPError(statusCode int, message string) error {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
	}
}

func badRequestError(message string) error {
	return newHTTPError(http.StatusBadRequest, message)
}

func unauthorizedError(message string) error {
	return newHTTPError(http.StatusUnauthorized, message)
}

func forbiddenError(message string) error {
	return newHTTPError(http.StatusForbidden, message)
}

func conflictError(message string) error {
	return newHTTPError(http.StatusConflict, message)
}

func serviceUnavailableError(message string) error {
	return newHTTPError(http.StatusServiceUnavailable, message)
}

func decodeRequest(r *http.Request, v any) error {
	_, err := decodeOptionalRequest(r, v)
	return err
}

func decodeOptionalRequest(r *http.Request, v any) (bool, error) {
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength == 0 {
		return false, nil
	}
	if err := render.DecodeJSON(r.Body, v); err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func sendJSONWithStatus(w http.ResponseWriter, statusCode int, status string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := Response{
		Status:  status,
		Message: message,
	}
	json.NewEncoder(w).Encode(resp)
}

func sendJSON(w http.ResponseWriter, status string, message string) {
	sendJSONWithStatus(w, http.StatusOK, status, message)
}

func sendError(w http.ResponseWriter, err error) {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		sendJSONWithStatus(w, httpErr.StatusCode, "error", httpErr.Message)
		return
	}

	sendJSONWithStatus(w, http.StatusInternalServerError, "error", err.Error())
}
