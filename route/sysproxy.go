package route

import (
	"net/http"
	"sparkle-service/log"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/xishang0128/sysproxy-go/sysproxy"
)

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
	log.Println("查询代理设置耗时：", time.Since(t))
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
	log.Println("设置 PAC 耗时：", time.Since(t), "\nURL:", req.Url)
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
	log.Println("设置代理耗时：", time.Since(t), "\nserver:", req.Server, "\nbypass:", req.Bypass)
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
	log.Println("禁用代理耗时：", time.Since(t))
	if err != nil {
		sendError(w, err)
		return
	}
	render.NoContent(w, r)
}
