package inventory

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/sksmith/smfg-inventory"
	"github.com/sksmith/smfg-inventory/api"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/render"
)

func Api (r chi.Router) {
	r.With(api.Paginate).Get("/", list)
	r.With(api.Paginate).Get("/search", search)

	r.Route("/{sku}", func(r chi.Router) {
		r.Use(productCtx)
		r.Put("/", update) // PUT /articles/123
	})

	r.Route("/reservation", func(r chi.Router) {
		r.Post("/", createReservation)

		r.Route("/{reservationID}", func(r chi.Router) {
			r.Use(reservationCtx)
			r.Delete("/", cancelReservation)
		})
	})
}

func list(w http.ResponseWriter, r *http.Request) {
	if err := render.RenderList(w, r, NewArticleListResponse(articles)); err != nil {
		render.Render(w, r, api.ErrRender(err))
		return
	}
}

func search(w http.ResponseWriter, r *http.Request) {
	if err := render.RenderList(w, r, NewArticleListResponse(articles)); err != nil {
		render.Render(w, r, api.ErrRender(err))
		return
	}
}

func productCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var product *Product
		var err error

		if sku := chi.URLParam(r, "sku"); sku != "" {
			product, err = dbGetArticle(articleID)
		} else if articleSlug := chi.URLParam(r, "articleSlug"); articleSlug != "" {
			product, err = dbGetArticleBySlug(articleSlug)
		} else {
			render.Render(w, r, api.ErrNotFound)
			return
		}
		if err != nil {
			render.Render(w, r, api.ErrNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), "article", article)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func update(w http.ResponseWriter, r *http.Request) {
	article := r.Context().Value("article").(*main.Article)

	data := &main.ArticleRequest{Article: article}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, main.ErrInvalidRequest(err))
		return
	}
	article = data.Article
	main.dbUpdateArticle(article.ID, article)

	render.Render(w, r, main.NewArticleResponse(article))
}

func cancelReservation(w http.ResponseWriter, r *http.Request) {
	article := r.Context().Value("article").(*main.Article)

	data := &main.ArticleRequest{Article: article}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, main.ErrInvalidRequest(err))
		return
	}
	article = data.Article
	main.dbUpdateArticle(article.ID, article)

	render.Render(w, r, main.NewArticleResponse(article))
}

func reservationCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var article *main.Article
		var err error

		if articleID := chi.URLParam(r, "articleID"); articleID != "" {
			article, err = main.dbGetArticle(articleID)
		} else if articleSlug := chi.URLParam(r, "articleSlug"); articleSlug != "" {
			article, err = main.dbGetArticleBySlug(articleSlug)
		} else {
			render.Render(w, r, main.ErrNotFound)
			return
		}
		if err != nil {
			render.Render(w, r, main.ErrNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), "article", article)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func createReservation(w http.ResponseWriter, r *http.Request) {
	article := r.Context().Value("article").(*main.Article)

	data := &main.ArticleRequest{Article: article}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, main.ErrInvalidRequest(err))
		return
	}
	article = data.Article
	main.dbUpdateArticle(article.ID, article)

	render.Render(w, r, main.NewArticleResponse(article))
}