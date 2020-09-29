package main

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/render"
)

func ListInventory(w http.ResponseWriter, r *http.Request) {

	// TODO Remove this, just added for testing
	// Sleep between 0 and 1000 milliseconds
	rand.Seed(time.Now().UnixNano())
	time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)

	if err := render.RenderList(w, r, NewArticleListResponse(articles)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// SearchArticles searches the Articles data for a matching article.
// It's just a stub, but you get the idea.
func SearchInventory(w http.ResponseWriter, r *http.Request) {
	render.RenderList(w, r, NewArticleListResponse(articles))
}

// UpdateArticle updates an existing Article in our persistent store.
func UpdateInventory(w http.ResponseWriter, r *http.Request) {
	article := r.Context().Value("article").(*Article)

	data := &ArticleRequest{Article: article}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	article = data.Article
	dbUpdateArticle(article.ID, article)

	render.Render(w, r, NewArticleResponse(article))
}
