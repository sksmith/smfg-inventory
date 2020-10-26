package inventory

import (
	"context"
	"errors"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/rs/zerolog/log"
	"github.com/sksmith/smfg-inventory/api"
	"net/http"
	"strconv"
)

const DefaultPageLimit = 50

type Api struct {
	service Service
}

func NewApi (service Service) *Api {
	return &Api{service: service}
}

func (a *Api) ConfigureRouter(r chi.Router) {
	r.With(api.Paginate).Get("/", a.List)
	r.With(api.Paginate).Get("/search", a.Search)
	r.Post("/", a.Create)

	r.Route("/{sku}", func(r chi.Router) {
		r.Use(a.ProductCtx)
		r.Post("/productionEvent", a.CreateProductionEvent)
	})

	r.Route("/reservation", func(r chi.Router) {
		r.Post("/", a.CreateReservation)

		r.Route("/{reservationID}", func(r chi.Router) {
			r.Use(a.ReservationCtx)
			r.Delete("/", a.CancelReservation)
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
	return nil
}

func (a *Api) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := getLimitAndOffset(r)
	if err != nil {
		api.Render(w, r, api.ErrInvalidRequest(err))
		return
	}

	products, err := a.service.GetAllProducts(r.Context(), limit, offset)
	if err != nil {
		log.Err(err).Send()
		api.Render(w, r, api.ErrInternalServerError())
		return
	}

	api.RenderList(w, r, NewProductListResponse(products))
}

func (a *Api) Create(w http.ResponseWriter, r *http.Request) {
	data := &CreateProductRequest{}
	if err := render.Bind(r, data); err != nil {
		api.Render(w, r, api.ErrInvalidRequest(err))
		return
	}

	if err := a.service.CreateProduct(r.Context(), *data.Product); err != nil {
		log.Err(err).Send()
		api.Render(w, r, api.ErrInternalServerError())
		return
	}
}

func (a *Api) Search(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := getLimitAndOffset(r)
	if err != nil {
		api.Render(w, r, api.ErrInvalidRequest(err))
		return
	}

	products, err := a.service.GetAllProducts(r.Context(), limit, offset)
	if err != nil {
		log.Err(err).Send()
		api.Render(w, r, api.ErrInternalServerError())
		return
	}

	api.RenderList(w, r, NewProductListResponse(products))
}

func (a *Api) ProductCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var product Product
		var err error

		if sku := chi.URLParam(r, "sku"); sku != "" {
			product, err = a.service.GetProduct(r.Context(), sku)
		} else {
			api.Render(w, r, api.ErrNotFound)
			return
		}
		if err != nil {
			api.Render(w, r, api.ErrNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), "product", product)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func NewProductListResponse(products []Product) []render.Renderer {
	var list []render.Renderer
	for _, product := range products {
		list = append(list, NewProductResponse(product))
	}
	return list
}

type CreateProductRequest struct {
	*Product
}

func (p *CreateProductRequest) Bind(_ *http.Request) error {
	if p.Upc == "" || p.Name == "" || p.Sku == "" {
		return errors.New("missing required field(s)")
	}

	return nil
}

type ProductionEventRequest struct {
	*ProductionEvent
}

func (p *ProductionEventRequest) Bind(_ *http.Request) error {
	if p.ProductionEvent == nil {
		return errors.New("missing required ProductionEvent fields")
	}

	return nil
}

type ProductionEventResponse struct {
	*ProductionEvent
}

func (p *ProductionEventResponse) Bind(_ *http.Request) error {
	if p.ProductionEvent == nil {
		return errors.New("missing required ProductionEvent fields")
	}

	return nil
}

func (a *Api) CreateProductionEvent(w http.ResponseWriter, r *http.Request) {
	product := r.Context().Value("product").(Product)

	data := &ProductionEventRequest{}
	if err := render.Bind(r, data); err != nil {
		api.Render(w, r, api.ErrInvalidRequest(err))
		return
	}

	if err := a.service.Produce(r.Context(), product, data.Quantity); err != nil {
		log.Err(err).Send()
		api.Render(w, r, api.ErrInternalServerError())
		return
	}

	return
}

func getLimitAndOffset(r *http.Request) (limit, offset int, err error) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit = DefaultPageLimit
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

func (a *Api) CancelReservation(_ http.ResponseWriter, _ *http.Request) {
	// Not implemented
	return
}

func (a *Api) ReservationCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Not implemented
		ctx := context.WithValue(r.Context(), "reservation", nil)
		next.ServeHTTP(w, r.WithContext(ctx))
		return
	})
}

func (a *Api) CreateReservation(_ http.ResponseWriter, _ *http.Request) {
	 //:= r.Context().Value("article").(*main.Article)
	// Not implemented
	return
}