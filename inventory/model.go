// This rudimentary model represents a fictional inventory tracking system for a factory. A real factory would obviously
// need much more fine grained detail and would probably use a different ubiquitous language.
//
// 0 = Product.Available + Product.Reserved + ShipmentDetail.Quantity - ProductionEvent.Quantity
package inventory

import (
	"context"
	"time"
)

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

type Service struct {
	repo Repository
}

func (s *Service) Produce(ctx context.Context, product Product, qty int64) error {
	tx, err := s.repo.BeginTransaction(ctx)
	if err != nil {
		return err
	}

	if err = tx.RollbackUnlessCommitted(ctx); err != nil {
		return err
	}

	evt := ProductionEvent{
		Sku:      product.Sku,
		Quantity: qty,
		Created:  time.Now(),
	}

	if err = s.repo.SaveProductionEvent(ctx, evt, tx); err != nil {
		return err
	}

	// Increase product available inventory
	product.Available += qty
	if err = s.repo.SaveProduct(ctx, product, tx); err != nil {
		return err
	}

	// Check reservations
	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetAllProducts(ctx context.Context, limit, offset int) ([]Product, error) {
	return s.repo.GetAllProducts(ctx, limit, offset)
}

func (s *Service) GetProduct(ctx context.Context, sku string) (Product, error) {
	return s.repo.GetProduct(ctx, sku)
}

// ProductionEvent is an entity. An addition to inventory through production of a Product.
type ProductionEvent struct {
	ID uint64
	Sku string
	Quantity int64
	Created time.Time
}

// Product is a value object. A SKU able to be produced by the factory.
type Product struct {
	Sku string
	Upc string
	Name string
	Available int64
	Reserved int64
}

// Reservation is an entity. An amount of inventory set aside for a given Customer.
type Reservation struct {
	ID uint64
	CustomerID uint64
	Details []ProductReservation
}

// ProductReservation is a value object. It's used for tracking inventory set aside for a Customer.
type ProductReservation struct {
	Sku string
	RequestedAmount int64
	ReservedAmount int64
}

// Shipment is an entity. An amount of inventory shipped to a Customer. All shipments of product should be tied to
// a reservation.
type Shipment struct {
	ID uint64
	ReservationID uint64
	Details []ShipmentDetail
}

// ShipmentDetail is an entity. It tracks quantities of Products on a given shipment.
type ShipmentDetail struct {
	ID uint64
	ShipmentID uint64
	Sku string
	Quantity int64
}

// Customer is an entity. A company or individual who pays for our services.
type Customer struct {
	ID uint64
	Name string
}