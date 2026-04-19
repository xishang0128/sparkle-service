package route

import (
	"fmt"
	"net/http"
	"time"

	"sparkle-service/log"
	appservice "sparkle-service/service"

	"github.com/go-chi/chi/v5"
)

var serviceController appservice.Controller

func serviceRouter() http.Handler {
	r := chi.NewRouter()

	r.Use(requestLogger)

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
		sendError(w, err)
		return
	}

	if action == "stop" && status == appservice.StatusStopped {
		sendJSON(w, "success", "服务已停止")
		return
	}

	if action == "restart" && status == appservice.StatusStopped {
		sendError(w, fmt.Errorf("服务未运行"))
		return
	}

	sendJSON(w, "success", message)

	go func() {
		time.Sleep(200 * time.Millisecond)
		if err := fn(); err != nil {
			log.Printf("%s 服务失败: %v", action, err)
		}
	}()
}
