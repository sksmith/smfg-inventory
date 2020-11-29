// This rudimentary model represents a fictional inventory tracking system for a factory. A real factory would obviously
// need much more fine grained detail and would probably use a different ubiquitous language.
package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/sksmith/bunnyq"
	"github.com/sksmith/smfg-inventory/db"
	"time"
)

func NewService(repo Repository, bq Queue) *service {
	return &service{repo: repo, bq: bq}
}

type Queue interface {
	Publish(ctx context.Context, exchange string, body []byte, options ...bunnyq.PublishOption) error
}

type Service interface {
	Produce(ctx context.Context, product Product, event *ProductionEvent) error
	Reserve(ctx context.Context, product Product, res *Reservation) error
	GetAllProducts(ctx context.Context, limit, offset int) ([]Product, error)
	GetProduct(ctx context.Context, sku string) (Product, error)
	CreateProduct(ctx context.Context, product Product) error
}

type service struct {
	repo Repository
	bq Queue
}

func (s *service) CreateProduct(ctx context.Context, product Product) error {
	err := s.repo.SaveProduct(ctx, product)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (s *service) Produce(ctx context.Context, product Product, event *ProductionEvent) error {
	if event == nil {
		return errors.New("event is required")
	}
	tx, err := s.repo.BeginTransaction(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	if event.RequestID == "" {
		return errors.New("request id is required")
	}
	if event.Quantity < 1 {
		return errors.New("quantity must be greater than zero")
	}

	dbEvent, err := s.repo.GetProductionEventByRequestID(ctx, event.RequestID, tx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return errors.WithStack(err)
	}

	if dbEvent.RequestID != "" {
		event.ID = dbEvent.ID
		event.Created = dbEvent.Created
		event.Quantity = dbEvent.Quantity
		event.RequestID = dbEvent.RequestID
		event.Sku = dbEvent.Sku
		return nil
	}

	event.Sku = product.Sku
	event.Created = time.Now()

	if err = s.repo.SaveProductionEvent(ctx, event, tx); err != nil {
		rollback(ctx, tx, err)
		return errors.WithMessage(err, "failed to save production event")
	}

	// Increase product available inventory
	product.Available += event.Quantity
	if err = s.repo.SaveProduct(ctx, product, tx); err != nil {
		rollback(ctx, tx, err)
		return errors.WithMessage(err, "failed to add production to product")
	}

	body, err := json.Marshal(product)
	if err != nil {
		rollback(ctx, tx, err)
		return errors.WithMessage(err, "failed to serialize message for queue")
	}
	if err = s.bq.Publish(ctx, "inventory.fanout", body); err != nil {
		rollback(ctx, tx, err)
		return errors.WithMessage(err, "failed to send inventory update to queue")
	}

	if err = tx.Commit(ctx); err != nil {
		rollback(ctx, tx, err)
		return errors.WithMessage(err, "failed to commit production transaction")
	}

	if err = s.fillReserves(ctx, product); err != nil {
		rollback(ctx, tx, err)
		return errors.WithMessage(err, "failed to fill reserves after production")
	}

	return nil
}

func (s *service) Reserve(ctx context.Context, pr Product, res *Reservation) error {
	tx, err := s.repo.BeginTransaction(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	dbRes, err := s.repo.GetReservationByRequestID(ctx, res.RequestID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if dbRes.RequestID != "" {
		res.RequestedQuantity = dbRes.RequestedQuantity
		res.Requester = dbRes.Requester
		res.Sku = dbRes.Sku
		res.RequestID = dbRes.RequestID
		res.ReservedQuantity = dbRes.ReservedQuantity
		res.State = dbRes.State
		res.ID = dbRes.ID
		res.Created = dbRes.Created
		return nil
	}

	res.State = Open
	res.Created = time.Now()

	if err = s.repo.SaveReservation(ctx, res, tx); err != nil {
		rollback(ctx, tx, err)
		return errors.WithStack(err)
	}

	if err = tx.Commit(ctx); err != nil {
		return errors.WithStack(err)
	}

	if err = s.fillReserves(ctx, pr); err != nil {
		rollback(ctx, tx, err)
		return errors.WithStack(err)
	}

	return nil
}

func (s *service) fillReserves(ctx context.Context, product Product) error {
	or, err := s.repo.GetSkuReservationsByState(ctx, product.Sku, Open, 100, 0)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, reservation := range or {
		if product.Available == 0 {
			break
		}

		remaining := reservation.RequestedQuantity - reservation.ReservedQuantity
		reserveAmount := remaining
		if remaining > product.Available {
			reserveAmount = product.Available
		}
		product.Available -= reserveAmount
		product.Reserved += reserveAmount
		reservation.ReservedQuantity += reserveAmount

		closed := false
		if reservation.ReservedQuantity == reservation.RequestedQuantity {
			if err := s.closeReservation(&product, &reservation); err != nil {
				return errors.WithStack(err)
			}
			closed = true
		}

		tx, err := s.repo.BeginTransaction(ctx)
		if err != nil {
			return errors.WithStack(err)
		}

		err = s.repo.SaveProduct(ctx, product, tx)
		if err != nil {
			rollback(ctx, tx, err)
			return errors.WithStack(err)
		}

		err = s.repo.UpdateReservation(ctx, reservation.ID, reservation.State, reservation.ReservedQuantity, tx)
		if err != nil {
			rollback(ctx, tx, err)
			return errors.WithStack(err)
		}

		if closed {
			body, err := json.Marshal(reservation)
			if err != nil {
				rollback(ctx, tx, err)
				return errors.WithMessage(err, "error marshalling reservation to send to queue")
			}
			err = s.bq.Publish(ctx, "reservation.filled.fanout", body)
			if err != nil {
				rollback(ctx, tx, err)
				return errors.WithStack(err)
			}
		}
		if err = tx.Commit(ctx); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (s *service) closeReservation(product *Product, reservation *Reservation) error {
	reservation.State = Closed
	product.Reserved -= reservation.RequestedQuantity
	return nil
}

func rollback(ctx context.Context, tx db.Transaction, err error) {
	rerr := tx.Rollback(ctx)
	if rerr != nil {
		log.Warn().Err(err).Msg("failed to rollback")
	}
}

func (s *service) getTx(ctx context.Context, txs ...db.Transaction) (tx db.Transaction, err error) {
	if len(txs) > 0 {
		tx = txs[0]
	} else {
		tx, err = s.repo.BeginTransaction(ctx)
	}
	return tx, err
}

func (s *service) GetAllProducts(ctx context.Context, limit, offset int) ([]Product, error) {
	return s.repo.GetAllProducts(ctx, limit, offset)
}

func (s *service) GetProduct(ctx context.Context, sku string) (Product, error) {
	product, err := s.repo.GetProduct(ctx, sku)
	if err != nil {
		return product, errors.WithStack(err)
	}
	return product, nil
}

// ProductionEvent is an entity. An addition to inventory through production of a Product.
type ProductionEvent struct {
	ID uint64 `json:"id"`
	RequestID string `json:"requestID"`
	Sku string `json:"sku"`
	Quantity int64 `json:"quantity"`
	Created time.Time `json:"created"`
}

// Product is a value object. A SKU able to be produced by the factory.
type Product struct {
	Sku string `json:"sku"`
	Upc string `json:"upc"`
	Name string `json:"name"`
	Available int64 `json:"available"`
	Reserved int64 `json:"reserved"`
}

type ReserveState string

const(
	Open ReserveState = "Open"
	Closed = "Closed"
	//None = ""
)

// Reservation is an entity. An amount of inventory set aside for a given Customer.
type Reservation struct {
	ID uint64 `json:"id"`
	RequestID string `json:"requestId"`
	Requester string `json:"requester"`
	Sku string `json:"sku"`
	State ReserveState `json:"state"`
	ReservedQuantity int64 `json:"reservedQuantity"`
	RequestedQuantity int64 `json:"requestedQuantity"`
	Created time.Time `json:"created"`
}
