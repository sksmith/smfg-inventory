package inventory

import (
	"context"
	"database/sql"
	"errors"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/rs/zerolog/log"
	"github.com/sksmith/smfg-inventory/api"
	"net/http"
	"strconv"
	"time"
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
	r.Post("/", a.Create)

	r.Route("/{sku}", func(r chi.Router) {
		r.Use(a.ProductCtx)
		r.Post("/productionEvent", a.CreateProductionEvent)

		r.Route("/reservation", func(r chi.Router) {
			r.Post("/", a.CreateReservation)

			r.Route("/{reservationID}", func(r chi.Router) {
				r.Use(a.ReservationCtx)
				r.Delete("/", a.CancelReservation)
			})
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

func (a *Api) ProductCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var product Product
		var err error

		sku := chi.URLParam(r, "sku")
		if sku == "" {
			api.Render(w, r, api.ErrInvalidRequest(errors.New("sku is required")))
			return
		}

		product, err = a.service.GetProduct(r.Context(), sku)

		if err != nil {
			if err == sql.ErrNoRows {
				api.Render(w, r, api.ErrNotFound)
			} else {
				log.Err(err).Str("sku", sku).Msg("error acquiring product")
				api.Render(w, r, api.ErrInternalServerError())
			}
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

	// we don't want to allow setting quantities upon creation of a product
	ProtectedReserved int `json:"reserved"`
	ProtectedAvailable int `json:"available"`
}

func (p *CreateProductRequest) Bind(_ *http.Request) error {
	if p.Upc == "" || p.Name == "" || p.Sku == "" {
		return errors.New("missing required field(s)")
	}

	return nil
}

type CreateProductionEventRequest struct {
	*ProductionEvent

	ProtectedID uint64 `json:"id"`
	ProtectedCreated time.Time `json:"created"`
}

func (p *CreateProductionEventRequest) Bind(_ *http.Request) error {
	if p.ProductionEvent == nil {
		return errors.New("missing required ProductionEvent fields")
	}

	return nil
}

type ProductionEventResponse struct {
	*ProductionEvent
}

func (p *ProductionEventResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

func (p *ProductionEventResponse) Bind(_ *http.Request) error {
	if p.ProductionEvent == nil {
		return errors.New("missing required ProductionEvent fields")
	}

	return nil
}

type ReservationRequest struct {
	*Reservation

	// ID is created by the database
	ProtectedID uint64 `json:"id"`

	// State is set automatically by the application
	ProtectedState string `json:"state"`

	// SKU is set through the URL
	ProtectedSku string `json:"sku"`

	// ReservedQuantity is calculated
	ProtectedReservedQuantity int64 `json:"reservedQuantity"`

	// Created is calculated
	ProtectedCreated time.Time `json:"created"`
}

func (r *ReservationRequest) Bind(_ *http.Request) error {
	if r.Reservation == nil {
		return errors.New("missing required Reservation fields")
	}

	return nil
}

type ReservationResponse struct {
	*Reservation
}

func (r *ReservationResponse) Bind(_ *http.Request) error {
	if r.Reservation == nil {
		return errors.New("missing required Reservation fields")
	}

	return nil
}

func (r *ReservationResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

func (a *Api) CreateProductionEvent(w http.ResponseWriter, r *http.Request) {
	product := r.Context().Value("product").(Product)

	data := &CreateProductionEventRequest{}
	if err := render.Bind(r, data); err != nil {
		api.Render(w, r, api.ErrInvalidRequest(err))
		return
	}

	if err := a.service.Produce(r.Context(), product, data.ProductionEvent); err != nil {
		log.Err(err).Send()
		api.Render(w, r, api.ErrInternalServerError())
		return
	}

	render.Status(r, http.StatusCreated)
	api.Render(w, r, &ProductionEventResponse{data.ProductionEvent})

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

func (a *Api) CreateReservation(w http.ResponseWriter, r *http.Request) {
	product := r.Context().Value("product").(Product)

	data := &ReservationRequest{}
	if err := render.Bind(r, data); err != nil {
		api.Render(w, r, api.ErrInvalidRequest(err))
		return
	}

	data.Sku = product.Sku

	err := a.service.Reserve(r.Context(), product, data.Reservation)
	if err != nil {
		log.Err(err).Send()
		api.Render(w, r, api.ErrInternalServerError())
	}

	resp := &ReservationResponse{Reservation: data.Reservation}
	render.Status(r, http.StatusCreated)
	api.Render(w, r, resp)

	return
}