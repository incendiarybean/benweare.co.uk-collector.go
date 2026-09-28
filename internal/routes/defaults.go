package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Redirect will peform an HTTP redirect to the given redirect Path.
func Redirect(redirectPath string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectPath, http.StatusFound)
	}
}

func DocsRouter() http.Handler {
	router := chi.NewRouter()
	router.Get("/swagger.json", func(response http.ResponseWriter, request *http.Request) {
		http.ServeFile(response, request, "api/swagger.json")
	})
	router.Get("/*", httpSwagger.Handler(httpSwagger.URL("/docs/swagger.json")))
	router.Get("/", Redirect("/docs/index.html"))

	return router
}
