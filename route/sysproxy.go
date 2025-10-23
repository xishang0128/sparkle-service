package route

import (
	"net/http"
	"sparkle-service/log"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/xishang0128/sysproxy-go/sysproxy"
)

type Request struct {
	Server string `json:"server"`
	Bypass string `json:"bypass"`
	Url    string `json:"url"`

	Device           string `json:"device,omitempty"`
	OnlyActiveDevice bool   `json:"only_active_device,omitempty"`
}

func httpProxyRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/status", status)
	r.Post("/pac", pac)
	r.Post("/proxy", proxy)
	r.Post("/disable", disable)
	return r
}

func status(w http.ResponseWriter, r *http.Request) {
	t := time.Now()
	status, err := sysproxy.QueryProxySettings("", true)
	log.Println("QueryProxySettings took:", time.Since(t))
	if err != nil {
		sendError(w, err)
		return
	}
	render.JSON(w, r, status)
}

func pac(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, err)
		return
	}

	t := time.Now()
	err := sysproxy.SetPac(req.Url, req.Device, req.OnlyActiveDevice)
	log.Println("SetPac took:", time.Since(t), "\nURL:", req.Url)
	if err != nil {
		sendError(w, err)
		return
	}
	render.NoContent(w, r)
}

func proxy(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, err)
		return
	}

	t := time.Now()
	err := sysproxy.SetProxy(req.Server, req.Bypass, req.Device, req.OnlyActiveDevice)
	log.Println("SetProxy took:", time.Since(t), "\nserver:", req.Server, "\nbypass:", req.Bypass)
	if err != nil {
		sendError(w, err)
		return
	}
	render.NoContent(w, r)
}

func disable(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := decodeRequest(r, &req); err != nil {
		sendError(w, err)
		return
	}

	t := time.Now()
	err := sysproxy.DisableProxy(req.Device, req.OnlyActiveDevice)
	log.Println("DisableProxy took:", time.Since(t))
	if err != nil {
		sendError(w, err)
		return
	}
	render.NoContent(w, r)
}

func decodeRequest(r *http.Request, v any) error {
	if r.ContentLength > 0 {
		return render.DecodeJSON(r.Body, v)
	}
	return nil
}
