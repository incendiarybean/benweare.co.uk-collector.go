package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/benweare.co.uk-api/internal/routes"
	"github.com/go-chi/chi/v5"
	"github.com/lmittmann/tint"
	_ "modernc.org/sqlite"
)

// @title			benweare.co.uk-api
// @version		1.0
// @description	This is an edge collector variant of the benweare.co.uk-api written in GO
// @contact.name	Benjamin Weare
// @contact.email	admin@benweare.co.uk
//
// @BasePath		/v1
func main() {
	w := os.Stderr

	// Set default slog values
	slog.SetDefault(slog.New(
		tint.NewTextHandler(w, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.RFC3339,
		})))

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Obtain port from ENV
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Configure a server
	server := &http.Server{
		Addr: fmt.Sprintf(":%s", port),
	}

	// Create or connect to a local storage
	db, err := sql.Open("sqlite", "./store.db")
	if err != nil {
		slog.Error("Could not obtain DB.")
		os.Exit(1)
	}

	// Create storage for articles and statistics
	// Statistics are to record when the collectors were previously ran, between reloads
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS articles (id TEXT primary key, hash TEXT, title TEXT, href TEXT, img TEXT, timestamp TEXT, outlet TEXT, description TEXT)"); err != nil {
		slog.Error("Could not obtain DB.")
		os.Exit(1)
	}

	slog.Info(fmt.Sprintf("Starting server with port: %s", port))

	go func() {
		// Register the available routes
		router := chi.NewRouter()
		router.Mount("/v1/news", routes.NewsRouter(db))
		router.Mount("/docs", routes.DocsRouter())

		server.Handler = router

		server.ListenAndServe()
	}()

	<-shutdown
}
