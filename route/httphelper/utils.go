package httphelper

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/render"
)

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

func NewError(statusCode int, message string) error {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
	}
}

func BadRequest(message string) error {
	return NewError(http.StatusBadRequest, message)
}

func Unauthorized(message string) error {
	return NewError(http.StatusUnauthorized, message)
}

func Forbidden(message string) error {
	return NewError(http.StatusForbidden, message)
}

func Conflict(message string) error {
	return NewError(http.StatusConflict, message)
}

func ServiceUnavailable(message string) error {
	return NewError(http.StatusServiceUnavailable, message)
}

func DecodeRequest(r *http.Request, v any) error {
	_, err := DecodeOptionalRequest(r, v)
	return err
}

func DecodeOptionalRequest(r *http.Request, v any) (bool, error) {
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

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func SendJSONWithStatus(w http.ResponseWriter, statusCode int, status string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := Response{
		Status:  status,
		Message: message,
	}
	json.NewEncoder(w).Encode(resp)
}

func SendJSON(w http.ResponseWriter, status string, message string) {
	SendJSONWithStatus(w, http.StatusOK, status, message)
}

func SendError(w http.ResponseWriter, err error) {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		SendJSONWithStatus(w, httpErr.StatusCode, "error", httpErr.Message)
		return
	}

	SendJSONWithStatus(w, http.StatusInternalServerError, "error", err.Error())
}
