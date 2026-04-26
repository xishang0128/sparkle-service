package serviceapi

import (
	"net/http"
	"time"

	"sparkle-service/log"
	"sparkle-service/route/httphelper"
	appservice "sparkle-service/service"

	"github.com/go-chi/chi/v5"
)

var serviceController appservice.Controller

func Router() http.Handler {
	r := chi.NewRouter()

	r.Use(httphelper.RequestLogger)

	r.Post("/stop", serviceStop)
	r.Post("/restart", serviceRestart)

	return r
}

func serviceStop(w http.ResponseWriter, r *http.Request) {
	controlServiceAsync(w, "stop", "服务停止中...", func() error { return serviceController.Stop() })
}

func serviceRestart(w http.ResponseWriter, r *http.Request) {
	controlServiceAsync(w, "restart", "服务重启中...", func() error { return serviceController.Restart() })
}

func controlServiceAsync(w http.ResponseWriter, action string, message string, fn func() error) {
	status, err := serviceController.Status()
	if err != nil {
		httphelper.SendError(w, err)
		return
	}

	if action == "stop" && status == appservice.StatusStopped {
		httphelper.SendJSON(w, "success", "服务已停止")
		return
	}

	if action == "restart" && status == appservice.StatusStopped {
		httphelper.SendError(w, httphelper.Conflict("服务未运行"))
		return
	}

	httphelper.SendJSONWithStatus(w, http.StatusAccepted, "success", message)

	go func() {
		time.Sleep(200 * time.Millisecond)
		if err := fn(); err != nil {
			log.Printf("%s 服务失败: %v", action, err)
		}
	}()
}
