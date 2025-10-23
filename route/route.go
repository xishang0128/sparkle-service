package route

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func router() *chi.Mux {
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Group(func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			sendJSON(w, "", "pong")
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			sendJSON(w, "", "auth success")
		})
		r.Mount("/sysproxy", httpProxyRouter())
		r.Mount("/core", coreManager())
	})
	return r
}
