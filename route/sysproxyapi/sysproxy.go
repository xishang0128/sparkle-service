package sysproxyapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"sparkle-service/log"
	"sparkle-service/route/httphelper"

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

func logSysproxyOperation(action, successMsg, failureMsg string, startedAt time.Time, err error, fields ...any) {
	duration := time.Since(startedAt)
	logFields := []any{
		"action", action,
		"success", err == nil,
		"duration", duration.String(),
		"duration_ms", float64(duration.Nanoseconds()) / float64(time.Millisecond),
	}
	logFields = append(logFields, fields...)

	if err != nil {
		logFields = append(logFields, "error", err.Error())
		log.S().Errorw(failureMsg, logFields...)
		return
	}
	log.S().Infow(successMsg, logFields...)
}

func sysproxyOptionLogFields(opts *sysproxy.Options) []any {
	if opts == nil {
		return nil
	}

	fields := []any{
		"only_active_device", opts.OnlyActiveDevice,
		"use_registry", opts.UseRegistry,
	}
	if opts.Device != "" {
		fields = append(fields, "device", opts.Device)
	}
	if opts.Proxy != "" {
		fields = append(fields, "server", opts.Proxy)
	}
	if opts.Bypass != "" {
		fields = append(fields, "bypass", splitBypassRules(opts.Bypass))
	}
	if opts.PACURL != "" {
		fields = append(fields, "url", opts.PACURL)
	}
	return fields
}

func splitBypassRules(bypass string) []string {
	parts := strings.Split(bypass, ",")
	rules := make([]string, 0, len(parts))
	for _, part := range parts {
		rule := strings.TrimSpace(part)
		if rule != "" {
			rules = append(rules, rule)
		}
	}
	return rules
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
	logSysproxyOperation("query_status", "查询代理设置完成", "查询代理设置失败", t, err, sysproxyOptionLogFields(opts)...)
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
	StopGuard()
	err := runSysproxyAsRequestUser(r, func() error {
		return runSysproxyMutation(func() error {
			if err := sysproxy.SetPac(opts); err != nil {
				return err
			}
			configureSysproxyGuardBestEffort(r, req.Guard, sysproxyGuardModePAC, opts)
			return nil
		})
	})
	logSysproxyOperation("set_pac", "设置 PAC 完成", "设置 PAC 失败", t, err, append(sysproxyOptionLogFields(opts), "guard", req.Guard)...)
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
	StopGuard()
	err := runSysproxyAsRequestUser(r, func() error {
		return runSysproxyMutation(func() error {
			if err := sysproxy.SetProxy(opts); err != nil {
				return err
			}
			configureSysproxyGuardBestEffort(r, req.Guard, sysproxyGuardModeProxy, opts)
			return nil
		})
	})
	logSysproxyOperation("set_proxy", "设置代理完成", "设置代理失败", t, err, append(sysproxyOptionLogFields(opts), "guard", req.Guard)...)
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
	StopGuard()
	err := runSysproxyAsRequestUser(r, func() error {
		return runSysproxyMutation(func() error {
			return sysproxy.DisableProxy(opts)
		})
	})
	logSysproxyOperation("disable_proxy", "禁用代理完成", "禁用代理失败", t, err, sysproxyOptionLogFields(opts)...)
	if err != nil {
		httphelper.SendError(w, err)
		return
	}
	render.NoContent(w, r)
}
