package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/benweare.co.uk-api/routes"
	"github.com/go-chi/chi"
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

	// Obtain a new logger
	logger := slog.New(slog.Default().Handler())

	db, err := sql.Open("sqlite", "./store.db")
	if err != nil {
		logger.Error("Could not obtain DB.")
		os.Exit(1)
	}

	db.Exec("CREATE TABLE IF NOT EXISTS articles (id TEXT primary key, title TEXT, url TEXT, img TEXT, date TEXT, name TEXT)")
	db.Exec("CREATE TABLE IF NOT EXISTS collector_stats (name TEXT primary key, timestamp INT)")

	// Obtain port from ENV
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info(fmt.Sprintf("Starting server with port: %s", port))

	router := chi.NewRouter()
	router.Mount("/v1/news", routes.NewsRouter(db))
	router.Mount("/swagger", routes.DocsRouter())

	go routes.NewsCollector(db)

	// Reassign port using string formatting
	port = fmt.Sprintf(":%s", port)

	http.ListenAndServe(port, router)
}
