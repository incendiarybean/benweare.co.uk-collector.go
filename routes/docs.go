package routes

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	httpSwagger "github.com/swaggo/http-swagger"
)

func DocsRouter() http.Handler {
	router := chi.NewRouter()
	router.Get("/doc.json", func(response http.ResponseWriter, request *http.Request) {
		http.ServeFile(response, request, "docs/swagger.json")
	})

	name, err := os.Hostname()
	if err != nil {
		slog.Error("Failed to get hostname to serve docs.")
		os.Exit(1)
	}

	host_address := fmt.Sprintf("http://%s:%s/swagger/doc.json", name, os.Getenv("PORT"))

	router.Get("/*", httpSwagger.Handler(httpSwagger.URL(host_address)))

	return router
}
