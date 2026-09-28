package routes

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Article struct {
	Id          string `json:"id"`
	Hash        string `json:"hash"`
	Outlet      string `json:"outlet"`
	Title       string `json:"title"`
	Img         string `json:"img,omitempty"`
	Href        string `json:"href,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
	Description string `json:"description,omitempty"`
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
		if err := row.Scan(&article.Id, &article.Hash, &article.Title, &article.Href, &article.Img, &article.Timestamp, &article.Outlet, &article.Description); err != nil {
			slog.Error(err.Error())
			http.Error(response, http.StatusText(422), 422)
			return
		}

		response.Header().Set("content-type", "application/json")

		json.NewEncoder(response).Encode(article)
	})
}

// GetOutlet godoc
//
//	@Summary		Obtain news articles by outlet
//	@Description	Obtain news articles by outlet
//	@Tags			News
//	@Produce		json
//	@Param			outlet	path		string	true	"Outlet Name"
//	@Success		200			{object}	Article
//	@Failure		400			{object}	string
//	@Failure		404			{object}	string
//	@Failure		500			{object}	string
//	@Router			/news/outlets/{outlet} [get]
func GetOutletsArticles(db *sql.DB) http.HandlerFunc {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		ctx := request.Context()

		outlet, ok := ctx.Value("outlet").(string)
		if !ok {
			http.Error(response, http.StatusText(422), 422)
			return
		}

		rows, err := db.Query("SELECT * FROM articles WHERE outlet=? COLLATE NOCASE", outlet)
		if err != nil {
			slog.Error("Could not query table.")
			http.Error(response, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}

		var articles []Article

		for rows.Next() {
			var article Article

			if err := rows.Scan(&article.Id, &article.Hash, &article.Title, &article.Href, &article.Img, &article.Timestamp, &article.Outlet, &article.Description); err != nil {
				slog.Error(err.Error())
				http.Error(response, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
				return
			}
			articles = append(articles, article)
		}

		if err = rows.Err(); err != nil {
			slog.Error(err.Error())
			http.Error(response, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}

		response.Header().Set("content-type", "application/json")

		json.NewEncoder(response).Encode(articles)
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
			http.Error(response, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}
		defer func() {
			_ = rows.Close()
		}()

		var articles []Article

		for rows.Next() {
			var article Article

			if err := rows.Scan(&article.Id, &article.Hash, &article.Title, &article.Href, &article.Img, &article.Timestamp, &article.Outlet, &article.Description); err != nil {
				slog.Error(err.Error())
				http.Error(response, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
				return
			}
			articles = append(articles, article)
		}

		if err = rows.Err(); err != nil {
			slog.Error(err.Error())
			http.Error(response, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}

		response.Header().Set("content-type", "application/json")

		if err = json.NewEncoder(response).Encode(articles); err != nil {
			slog.Error(err.Error())
			http.Error(response, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}
	})
}

// Obtain contextual values from the path, e.g. ArticleId
func newsContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		// Define param names
		articleId := chi.URLParam(request, "articleId")
		outletName := chi.URLParam(request, "outlet")

		// Add params to context
		ctx := context.WithValue(request.Context(), "article", articleId)
		ctx = context.WithValue(ctx, "outlet", outletName)

		// Provide the context to the next stage
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

		router.Route("/outlets", func(router chi.Router) {
			router.Use(newsContext)

			router.Get("/", func(response http.ResponseWriter, request *http.Request) {
				rows, err := db.Query("SELECT DISTINCT outlet FROM articles;")
				if err != nil {
					slog.Error("Could not query table.")
					http.Error(response, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
					return
				}

				var articles []string

				for rows.Next() {
					var article string

					if err := rows.Scan(&article); err != nil {
						slog.Error(err.Error())
						http.Error(response, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
						return
					}
					articles = append(articles, article)
				}

				// slog.Info(strings.Join(outlets, ""))

				if err = rows.Err(); err != nil {
					slog.Error(err.Error())
					http.Error(response, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
					return
				}
				response.Header().Set("content-type", "application/json")

				if err = json.NewEncoder(response).Encode(articles); err != nil {
					slog.Error(err.Error())
					http.Error(response, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
					return
				}
			})

			router.Get("/{outlet}", GetOutletsArticles(db))
		})

	})
	return router
}
