// This rudimentary model represents a fictional inventory tracking system for a factory. A real factory would obviously
// need much more fine grained detail and would probably use a different ubiquitous language.
package inventory

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/sksmith/smfg-inventory/db"
	"time"
)

func NewService(repo Repository, queue Queue) *service {
	return &service{repo: repo, queue: queue}
}

type Service interface {
	Produce(ctx context.Context, product Product, qty int64) error
	Reserve(ctx context.Context, product Product, res *Reservation) error
	GetAllProducts(ctx context.Context, limit, offset int) ([]Product, error)
	GetProduct(ctx context.Context, sku string) (Product, error)
	CreateProduct(ctx context.Context, product Product) error
}

type service struct {
	repo Repository
	queue Queue
}

func (s *service) CreateProduct(ctx context.Context, product Product) error {
	return s.repo.SaveProduct(ctx, product)
}

func (s *service) Produce(ctx context.Context, product Product, qty int64) error {
	tx, err := s.repo.BeginTransaction(ctx)
	if err != nil {
		return err
	}

	evt := ProductionEvent{
		Sku:      product.Sku,
		Quantity: qty,
		Created:  time.Now(),
	}

	if err = s.repo.SaveProductionEvent(ctx, evt, tx); err != nil {
		rollback(ctx, tx, err)
		return err
	}

	// Increase product available inventory
	product.Available += qty
	if err = s.repo.SaveProduct(ctx, product, tx); err != nil {
		rollback(ctx, tx, err)
		return err
	}

	if err = s.queue.Send(product, Exchange("inventory.fanout")); err != nil {
		rollback(ctx, tx, err)
		return err
	}

	if err = s.fillReserves(ctx, product); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *service) Reserve(ctx context.Context, pr Product, res *Reservation) error {
	tx, err := s.repo.BeginTransaction(ctx)
	if err != nil {
		return err
	}

	res.State = Open
	res.Created = time.Now()

	if err = s.repo.SaveReservation(ctx, res, tx); err != nil {
		rollback(ctx, tx, err)
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	if err = s.fillReserves(ctx, pr); err != nil {
		rollback(ctx, tx, err)
		return err
	}

	return nil
}

func (s *service) fillReserves(ctx context.Context, product Product) error {
	or, err := s.repo.GetSkuReservesByState(ctx, product.Sku, Open, 100, 0)
	if err != nil {
		return err
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
				return err
			}
			closed = true
		}

		tx, err := s.repo.BeginTransaction(ctx)
		if err != nil {
			return err
		}

		err = s.repo.SaveProduct(ctx, product, tx)
		if err != nil {
			rollback(ctx, tx, err)
			return err
		}

		err = s.repo.SaveReservation(ctx, &reservation, tx)
		if err != nil {
			rollback(ctx, tx, err)
			return err
		}

		if closed {
			err := s.queue.Send(reservation, Exchange("reservation.filled.fanout"))
			if err != nil {
				rollback(ctx, tx, err)
				return err
			}
		}
		if err = tx.Commit(ctx); err != nil {
			return err
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
	return s.repo.GetProduct(ctx, sku)
}

// ProductionEvent is an entity. An addition to inventory through production of a Product.
type ProductionEvent struct {
	ID uint64 `json:"id"`
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
)

// Reservation is an entity. An amount of inventory set aside for a given Customer.
type Reservation struct {
	ID uint64
	Requester string
	Sku string
	State ReserveState
	ReservedQuantity int64
	RequestedQuantity int64
	Created time.Time `json:"created"`
}
