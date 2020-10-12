package inventory

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/sksmith/smfg-inventory/api"
	"net/http"
	"strconv"

	"github.com/go-chi/render"
)

var service *Service

func WithService(s *Service) {
	service = s
}

func Api (r chi.Router) {
	r.With(api.Paginate).Get("/", list)
	r.With(api.Paginate).Get("/search", search)

	r.Route("/{sku}", func(r chi.Router) {
		r.Use(productCtx)
		//r.Put("/", update) // PUT /articles/123F
	})

	r.Route("/reservation", func(r chi.Router) {
		r.Post("/", createReservation)

		r.Route("/{reservationID}", func(r chi.Router) {
			r.Use(reservationCtx)
			r.Delete("/", cancelReservation)
		})
	})
}

type ProductResponse struct {
	Product
}

func NewProductResponse(product Product) *ProductResponse {
	resp := &ProductResponse{Product: product}
	return resp
}

func (rd *ProductResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	// Pre-processing before a response is marshalled and sent across the wire
	// This doesn't do much now, but I wanted to document the pattern easily here
	return nil
}

func NewProductListResponse(products []Product) []render.Renderer {
	var list []render.Renderer
	for _, product := range products {
		list = append(list, NewProductResponse(product))
	}
	return list
}

func list(w http.ResponseWriter, r *http.Request) {

	limit, offset, err := getLimitAndOffset(r)
	if err != nil {
		// TODO - Should log these exceptions
		_ = render.Render(w, r, api.ErrInvalidRequest(err))
		return
	}

	products, err := service.GetAllProducts(r.Context(), limit, offset)
	if err != nil {
		_ = render.Render(w, r, api.ErrRender(err))
		return
	}

	if err := render.RenderList(w, r, NewProductListResponse(products)); err != nil {
		_ = render.Render(w, r, api.ErrRender(err))
		return
	}
}

func getLimitAndOffset(r *http.Request) (limit, offset int, err error) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return 0, 0, err
		}
	}

	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			return 0, 0, err
		}
	}

	return limit, offset, nil
}

func search(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := getLimitAndOffset(r)
	if err != nil {
		_ = render.Render(w, r, api.ErrRender(err))
		return
	}

	products, err := service.GetAllProducts(r.Context(), limit, offset)
	if err != nil {
		_ = render.Render(w, r, api.ErrRender(err))
		return
	}

	if err := render.RenderList(w, r, NewProductListResponse(products)); err != nil {
		_ = render.Render(w, r, api.ErrRender(err))
		return
	}
}

func productCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var product Product
		var err error

		if sku := chi.URLParam(r, "sku"); sku != "" {
			product, err = service.GetProduct(r.Context(), sku)
		} else {
			_ = render.Render(w, r, api.ErrNotFound)
			return
		}
		if err != nil {
			_ = render.Render(w, r, api.ErrNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), "product", product)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func cancelReservation(_ http.ResponseWriter, _ *http.Request) {
	// Not implemented
	return
}

func reservationCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Not implemented
		ctx := context.WithValue(r.Context(), "reservation", nil)
		next.ServeHTTP(w, r.WithContext(ctx))
		return
	})
}

func createReservation(_ http.ResponseWriter, _ *http.Request) {
	 //:= r.Context().Value("article").(*main.Article)
	// Not implemented
	return
}