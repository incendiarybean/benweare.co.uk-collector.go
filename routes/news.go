package routes

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi"
)

type Article struct {
	Id    string `json:"id"`
	Url   string `json:"url"`
	Title string `json:"title"`
	Img   string `json:"img"`
	Date  string `json:"date"`
	Name  string `json:"name"`
}

type ResponseItems struct {
	Items []Article
}

type ResponseLink struct {
	Action string
	Href   string
}

type ApiResponse struct {
	Response    ResponseItems
	Description string
	Timestamp   string
	Link        ResponseLink
}

type CollectorStats struct {
	Name      string
	Timestamp sql.NullInt64
}

func NewsCollector(db *sql.DB) {

	row := db.QueryRow("SELECT * FROM collector_stats WHERE `name`='news';")

	stats := &CollectorStats{}
	err := row.Scan(&stats.Name, &stats.Timestamp)
	if err != nil {
		if err != sql.ErrNoRows {
			slog.Error(fmt.Sprintf("Error obtaining previous collection date: %s", err))
			return
		}

		slog.Debug("No collector stats available for News yet...")
		db.Exec("INSERT INTO collector_stats (name, timestamp) VALUES (?, NULL)", "news")
	}

	for {
		time_diff := time.Now().UnixMilli() - stats.Timestamp.Int64

		var last_collected string
		if stats.Timestamp.Valid {
			last_collected = string(time.UnixMilli(stats.Timestamp.Int64).Format(time.RFC3339))
		} else {
			last_collected = "uncollected"
		}

		slog.Debug(fmt.Sprintf("News was last collected at: %s - %s(s) ago.", last_collected, fmt.Sprint(time_diff/1000)))
		if stats.Timestamp.Int64 == 0 || time_diff > (5*time.Minute).Milliseconds() {
			slog.Debug("News collection has expired, running collector...")

			slog.Debug("Requesting news collection from Node...")

			client := &http.Client{}
			url := "https://benweare.co.uk/api/news/articles"
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				slog.Error(fmt.Sprintf("Error building request: %s", err))
				continue
			}

			req.Header.Add("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				slog.Error(fmt.Sprintf("Error making request: %s", err))
				continue
			}

			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				slog.Debug(string(body))
				slog.Error(fmt.Sprintf("Error reading response body: %s", err))
				continue
			}

			var net ApiResponse
			json_err := json.Unmarshal(body, &net)
			if json_err != nil {
				slog.Debug(string(body))
				slog.Error(fmt.Sprintf("Error converting JSON to API Response: %s", err))
				return
			}

			statement, err := db.Prepare("INSERT INTO articles (id, title, url, img, date, name) VALUES (?, ?, ?, ?, ?, ?)")
			if err != nil {
				slog.Error(fmt.Sprintf("Error generating prepared statement: %s", err))
				return
			}

			for index := range net.Response.Items {
				article := net.Response.Items[index]

				statement.Query(article.Id, article.Title, article.Url, article.Img, article.Date, article.Name)
			}

			db.Exec("UPDATE collector_stats SET timestamp=? WHERE name=?", time.Now().UnixMilli(), "news")

			slog.Debug("News collection collected and processed.")
		} else {
			slog.Debug("News collection has not yet expired, skipping.")
		}

		time.Sleep(5 * time.Minute) // 5 minutes
	}
}

// GetArticle godoc
//
//	@Summary		Obtain a news article by ID
//	@Description	get article by ID
//	@Tags			News
//	@Produce		json
//	@Param			articleId	path		string	true	"Article ID"
//	@Success		200			{object}	Article
//	@Failure		400			{object}	string
//	@Failure		404			{object}	string
//	@Failure		500			{object}	string
//	@Router			/news/{articleId} [get]
func GetArticle(db *sql.DB) http.HandlerFunc {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		articleId, ok := ctx.Value("article").(string)
		if !ok {
			http.Error(response, http.StatusText(422), 422)
			return
		}

		row := db.QueryRow("SELECT * FROM articles WHERE id=?", articleId)

		article := &Article{}
		err := row.Scan(&article.Id, &article.Title, &article.Url, &article.Img, &article.Date, &article.Name)
		if err != nil {
			slog.Error(err.Error())
			http.Error(response, http.StatusText(422), 422)
			return
		}

		response.Header().Set("content-type", "application/json")

		json.NewEncoder(response).Encode(article)
	})
}

// ListArticle godoc
//
//	@Summary		Show all articles
//	@Description	get all articles
//	@Tags			News
//	@Produce		json
//	@Success		200	{object}	Article
//	@Failure		400	{object}	string
//	@Failure		404	{object}	string
//	@Failure		500	{object}	string
//	@Router			/news [get]
func ListArticles(db *sql.DB) http.HandlerFunc {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		rows, err := db.Query("SELECT * FROM articles;")
		if err != nil {
			slog.Error("Could not query table.")
			http.Error(response, http.StatusText(502), 502)
			return
		}
		defer rows.Close()

		var articles []Article

		for rows.Next() {
			var article Article

			if err := rows.Scan(&article.Id, &article.Title, &article.Url, &article.Img, &article.Date, &article.Name); err != nil {
				slog.Error(err.Error())
				http.Error(response, http.StatusText(502), 502)
				return
			}
			articles = append(articles, article)
		}

		if err = rows.Err(); err != nil {
			slog.Error(err.Error())
			http.Error(response, http.StatusText(502), 502)
			return
		}

		response.Header().Set("content-type", "application/json")

		json.NewEncoder(response).Encode(articles)
	})
}

// Obtain contextual values from the path, e.g. ArticleId
func newsContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		articleId := chi.URLParam(request, "articleId")
		ctx := context.WithValue(request.Context(), "article", articleId)
		next.ServeHTTP(response, request.WithContext(ctx))
	})
}

// Generate routes for the News processing
func NewsRouter(db *sql.DB) http.Handler {
	router := chi.NewRouter()
	router.Route("/", func(router chi.Router) {
		router.Get("/", ListArticles(db)) // GET /news

		router.Route("/{articleId}", func(router chi.Router) {
			router.Use(newsContext)

			router.Get("/", GetArticle(db))
		})

	})
	return router
}
