package inventory

import (
	"context"
	"github.com/sksmith/smfg-inventory/db"
)

type Repository interface {
	SaveProductionEvent(ctx context.Context, event ProductionEvent, tx ...db.Transaction) error
	SaveReservation(ctx context.Context, reservation *Reservation, tx ...db.Transaction) error
	GetSkuReservesByState(ctx context.Context, sku string, state ReserveState, limit, offset int, tx ...db.Transaction) ([]Reservation, error)
	SaveProduct(ctx context.Context, product Product, tx ...db.Transaction) error
	GetProduct(ctx context.Context, sku string, tx ...db.Transaction) (Product, error)
	GetAllProducts(ctx context.Context, limit int, offset int, tx ...db.Transaction) ([]Product, error)
	BeginTransaction(ctx context.Context) (db.Transaction, error)
}
