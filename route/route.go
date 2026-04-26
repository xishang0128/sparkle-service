package route

import (
	"net/http"
	"sparkle-service/route/auth"
	"sparkle-service/route/coreapi"
	"sparkle-service/route/httphelper"
	"sparkle-service/route/serviceapi"
	"sparkle-service/route/sysapi"
	"sparkle-service/route/sysproxyapi"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func router() *chi.Mux {
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Group(func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			httphelper.SendJSON(w, "success", "pong")
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			httphelper.SendJSON(w, "success", "auth success")
		})
		r.Mount("/service", serviceapi.Router())
		r.Mount("/sysproxy", sysproxyapi.Router())
		r.Mount("/core", coreapi.Router())
		r.Mount("/sys", sysapi.Router())
	})
	return r
}
