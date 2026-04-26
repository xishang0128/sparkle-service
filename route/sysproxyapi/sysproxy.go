package sysproxyapi

import (
	"fmt"
	"net/http"
	"sparkle-service/log"
	"sparkle-service/route/httphelper"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/xishang0128/sysproxy-go/sysproxy"
)

type proxyRequest struct {
	Server string `json:"server,omitempty"`
	Bypass string `json:"bypass,omitempty"`
	Url    string `json:"url,omitempty"`

	Device           string `json:"device,omitempty"`
	OnlyActiveDevice bool   `json:"only_active_device,omitempty"`
	UseRegistry      bool   `json:"use_registry,omitempty"`
	Guard            bool   `json:"guard,omitempty"`
}

func Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/status", status)
	r.Get("/events", sysproxyEvents)
	r.Post("/pac", pac)
	r.Post("/proxy", proxy)
	r.Post("/disable", disable)
	return r
}

func status(w http.ResponseWriter, r *http.Request) {
	t := time.Now()
	opts := prepareSysproxyOptions(r, &sysproxy.Options{OnlyActiveDevice: true})
	var status any
	err := runSysproxyAsRequestUser(r, func() error {
		result, err := sysproxy.QueryProxySettings(opts)
		if err != nil {
			return err
		}
		status = result
		return nil
	})
	log.Println("查询代理设置耗时：", time.Since(t))
	if err != nil {
		httphelper.SendError(w, err)
		return
	}
	render.JSON(w, r, status)
}

func pac(w http.ResponseWriter, r *http.Request) {
	var req proxyRequest
	if err := httphelper.DecodeRequest(r, &req); err != nil {
		httphelper.SendError(w, httphelper.BadRequest(fmt.Sprintf("无效的请求体: %v", err)))
		return
	}

	t := time.Now()
	opts := prepareSysproxyOptions(r, &sysproxy.Options{
		PACURL:           req.Url,
		Device:           req.Device,
		OnlyActiveDevice: req.OnlyActiveDevice,
		UseRegistry:      req.UseRegistry,
	})
	err := runSysproxyAsRequestUser(r, func() error {
		if err := sysproxy.SetPac(opts); err != nil {
			return err
		}
		return configureSysproxyGuard(r, req.Guard, sysproxyGuardModePAC, opts)
	})
	log.Println("设置 PAC 耗时：", time.Since(t), "\nURL:", req.Url)
	if err != nil {
		httphelper.SendError(w, err)
		return
	}
	render.NoContent(w, r)
}

func proxy(w http.ResponseWriter, r *http.Request) {
	var req proxyRequest
	if err := httphelper.DecodeRequest(r, &req); err != nil {
		httphelper.SendError(w, httphelper.BadRequest(fmt.Sprintf("无效的请求体: %v", err)))
		return
	}

	t := time.Now()
	opts := prepareSysproxyOptions(r, &sysproxy.Options{
		Proxy:            req.Server,
		Bypass:           req.Bypass,
		Device:           req.Device,
		OnlyActiveDevice: req.OnlyActiveDevice,
		UseRegistry:      req.UseRegistry,
	})
	err := runSysproxyAsRequestUser(r, func() error {
		if err := sysproxy.SetProxy(opts); err != nil {
			return err
		}
		return configureSysproxyGuard(r, req.Guard, sysproxyGuardModeProxy, opts)
	})
	log.Println("设置代理耗时：", time.Since(t), "\nserver:", req.Server, "\nbypass:", req.Bypass)
	if err != nil {
		httphelper.SendError(w, err)
		return
	}
	render.NoContent(w, r)
}

func disable(w http.ResponseWriter, r *http.Request) {
	var req proxyRequest
	if err := httphelper.DecodeRequest(r, &req); err != nil {
		httphelper.SendError(w, httphelper.BadRequest(fmt.Sprintf("无效的请求体: %v", err)))
		return
	}

	t := time.Now()
	opts := prepareSysproxyOptions(r, &sysproxy.Options{
		Device:           req.Device,
		OnlyActiveDevice: req.OnlyActiveDevice,
		UseRegistry:      req.UseRegistry,
	})
	err := runSysproxyAsRequestUser(r, func() error {
		if err := sysproxy.DisableProxy(opts); err != nil {
			return err
		}
		StopGuard()
		return nil
	})
	log.Println("禁用代理耗时：", time.Since(t))
	if err != nil {
		httphelper.SendError(w, err)
		return
	}
	render.NoContent(w, r)
}
