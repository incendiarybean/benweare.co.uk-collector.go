package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/benweare.co.uk-api/internal/routes"
	"github.com/go-chi/chi/v5"
	"github.com/lmittmann/tint"
	"golang.org/x/net/html"
	_ "modernc.org/sqlite"
)

// func getElementByAttr()

func getNews() {
	response, err := http.DefaultClient.Get("https://www.pcgamer.com/uk/news/")
	if err != nil {
		slog.Error(fmt.Sprintf("Error: %s", err.Error()))
	}

	doc := html.NewTokenizer(response.Body)
	if err != nil {
		slog.Error(fmt.Sprintf("Error: %s", err.Error()))
	}

	for {
		if doc.Next() == html.ErrorToken {
			break
		}

		token := doc.Token()
		attributes := token.Attr

		for _, attr := range attributes {
			if attr.Key == "class" {
				classes := strings.Split(attr.Val, " ")
				if slices.Contains(classes, "listingResult") {
					slog.Info(fmt.Sprintf("%s", attr.Val))
				}
			}
		}
	}
}

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

	// Create or connect to a local storage
	db, err := sql.Open("sqlite", "./store.db")
	if err != nil {
		logger.Error("Could not obtain DB.")
		os.Exit(1)
	}

	// Create storage for articles and statistics
	// Statistics are to record when the collectors were previously ran, between reloads
	db.Exec("CREATE TABLE IF NOT EXISTS collector_stats (name TEXT primary key, timestamp INT)")
	db.Exec("CREATE TABLE IF NOT EXISTS articles (id TEXT primary key, title TEXT, url TEXT, img TEXT, date TEXT, name TEXT)")

	// Obtain port from ENV
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info(fmt.Sprintf("Starting server with port: %s", port))

	// Register the available routes
	router := chi.NewRouter()
	router.Mount("/v1/news", routes.NewsRouter(db))
	router.Mount("/swagger", routes.DocsRouter())

	getNews()
	// go routes.NewsCollector(db)

	// // Reassign port using string formatting
	// port = fmt.Sprintf(":%s", port)

	// http.ListenAndServe(port, router)
}
