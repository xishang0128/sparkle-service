package route

import (
	"net/http"
	"sparkle-service/core"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

var (
	cm     *core.CoreManager
	isInit atomic.Bool
)

func coreManagerRouter() http.Handler {
	if !isInit.Load() {
		cm = core.NewCoreManager(core.WithTrafficMonitorPipeSDDL(trafficMonitorPipeSDDL()))
		isInit.Store(true)
	}

	r := chi.NewRouter()

	r.Use(requestLogger)

	r.Get("/", coreStatus)
	r.Get("/events", coreEvents)
	r.HandleFunc("/controller", coreControllerProxy)
	r.HandleFunc("/controller/*", coreControllerProxy)
	r.Get("/profile", coreProfile)
	r.Post("/profile", coreSaveProfile)
	r.Patch("/profile", corePatchProfile)
	r.Post("/start", coreStart)
	r.Post("/stop", coreStop)
	r.Post("/restart", coreRestart)

	return r
}

func trafficMonitorPipeSDDL() string {
	if sid, ok := GetKeyManager().GetAuthorizedSID(); ok {
		return "D:PAI(A;OICI;GWGR;;;" + sid + ")(A;OICI;GWGR;;;SY)"
	}
	return ""
}

func stopCoreManager() error {
	if !isInit.Load() || cm == nil {
		return nil
	}
	return cm.StopCore()
}

func coreStatus(w http.ResponseWriter, r *http.Request) {
	status, err := cm.GetProcessInfo()
	if err != nil {
		sendError(w, err)
		return
	}
	render.JSON(w, r, status)
}

func coreProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := core.LoadLaunchProfile()
	if err != nil {
		sendError(w, err)
		return
	}
	render.JSON(w, r, profile)
}

func coreSaveProfile(w http.ResponseWriter, r *http.Request) {
	var profile core.LaunchProfile
	if err := decodeRequest(r, &profile); err != nil {
		sendError(w, badRequestError(err.Error()))
		return
	}

	if err := core.SaveLaunchProfile(profile); err != nil {
		sendError(w, badRequestError(err.Error()))
		return
	}
	normalized, err := core.LoadLaunchProfile()
	if err != nil {
		sendError(w, badRequestError(err.Error()))
		return
	}
	cm.ApplyLaunchProfile(normalized, coreLaunchOptions(r)...)

	sendJSON(w, "success", "核心启动配置已更新")
}

func corePatchProfile(w http.ResponseWriter, r *http.Request) {
	var patch core.LaunchProfilePatch
	if err := decodeRequest(r, &patch); err != nil {
		sendError(w, badRequestError(err.Error()))
		return
	}

	profile, err := core.PatchLaunchProfile(patch)
	if err != nil {
		sendError(w, badRequestError(err.Error()))
		return
	}
	cm.ApplyLaunchProfile(profile, coreLaunchOptions(r)...)

	sendJSON(w, "success", "核心启动配置已更新")
}

func coreStart(w http.ResponseWriter, r *http.Request) {
	profile, hasProfile, err := decodeOptionalLaunchProfile(r)
	if err != nil {
		sendError(w, badRequestError(err.Error()))
		return
	}
	if hasProfile {
		if err := core.SaveLaunchProfile(*profile); err != nil {
			sendError(w, badRequestError(err.Error()))
			return
		}
	}

	if err := cm.StartCoreWithProfile(profile, coreLaunchOptions(r)...); err != nil {
		sendError(w, err)
		return
	}

	sendCoreReady(w, r, "核心启动成功")
}

func coreStop(w http.ResponseWriter, r *http.Request) {
	if err := cm.StopCore(); err != nil {
		sendError(w, err)
		return
	}
	sendJSON(w, "success", "核心停止成功")
}

func coreRestart(w http.ResponseWriter, r *http.Request) {
	profile, hasProfile, err := decodeOptionalLaunchProfile(r)
	if err != nil {
		sendError(w, badRequestError(err.Error()))
		return
	}
	if hasProfile {
		if err := core.SaveLaunchProfile(*profile); err != nil {
			sendError(w, badRequestError(err.Error()))
			return
		}
	}

	if err := cm.RestartCoreWithProfile(profile, coreLaunchOptions(r)...); err != nil {
		sendError(w, err)
		return
	}
	sendCoreReady(w, r, "核心重启成功")
}

func decodeOptionalLaunchProfile(r *http.Request) (*core.LaunchProfile, bool, error) {
	var profile core.LaunchProfile
	ok, err := decodeOptionalRequest(r, &profile)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	return &profile, true, nil
}

func sendCoreReady(w http.ResponseWriter, r *http.Request, message string) {
	status, err := cm.GetProcessInfo()
	if err != nil {
		sendJSON(w, "success", message)
		return
	}

	render.JSON(w, r, map[string]any{
		"status":  "success",
		"message": message,
		"core":    status,
	})
}
