package sysapi

import (
	"fmt"
	"net/http"
	"sparkle-service/route/httphelper"
	"sparkle-service/sys"

	"github.com/go-chi/chi/v5"
)

type dnsRequest struct {
	Device  string   `json:"device,omitempty"`
	Servers []string `json:"servers,omitempty"`
}

func Router() http.Handler {
	r := chi.NewRouter()

	r.Post("/dns/set", setDns)

	return r
}

func setDns(w http.ResponseWriter, r *http.Request) {
	var req dnsRequest
	if err := httphelper.DecodeRequest(r, &req); err != nil {
		httphelper.SendError(w, httphelper.BadRequest(fmt.Sprintf("无效的请求体: %v", err)))
		return
	}
	if err := sys.SetDns(req.Device, req.Servers); err != nil {
		httphelper.SendError(w, err)
		return
	}
	httphelper.SendJSON(w, "success", "DNS 设置成功")
}
