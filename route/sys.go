package route

import (
	"fmt"
	"net/http"
	"sparkle-service/sys"

	"github.com/go-chi/chi/v5"
)

func sysRouter() http.Handler {
	r := chi.NewRouter()

	r.Post("/dns/set", setDns)

	return r
}

func setDns(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, badRequestError(fmt.Sprintf("无效的请求体: %v", err)))
		return
	}
	if err := sys.SetDns(req.Device, req.Servers); err != nil {
		sendError(w, err)
		return
	}
	sendJSON(w, "success", "DNS 设置成功")
}
