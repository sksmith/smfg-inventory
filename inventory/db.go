package inventory

import (
	"context"
	"errors"
	"github.com/sksmith/smfg-inventory/db"
)

type Repository interface {
	SaveProductionEvent(ctx context.Context, event ProductionEvent, tx ...db.Transaction) error
	SaveProduct(ctx context.Context, product Product, tx ...db.Transaction) error
	GetProduct(ctx context.Context, sku string, tx ...db.Transaction) (Product, error)
	GetAllProducts(ctx context.Context, limit int, offset int, tx ...db.Transaction) ([]Product, error)
	BeginTransaction(ctx context.Context) (db.Transaction, error)
}

type memoryRepo struct {
	products []Product
	events []ProductionEvent
}

func NewMemoryRepo() Repository {
	return &memoryRepo{
		products: []Product{
			Product{
				Sku:       "SSPROCK01",
				Upc:       "10235668",
				Name:      "Small Basic Sprocket",
				Available: 6,
				Reserved:  0,
			},
			Product{
				Sku:       "SSPROCK02",
				Upc:       "1255506827",
				Name:      "Small Advanced Sprocket",
				Available: 3,
				Reserved:  0,
			},
			Product{
				Sku:       "LSPROCK01",
				Upc:       "4670235668",
				Name:      "Large Simple Sprocket",
				Available: 2,
				Reserved:  0,
			},
		},
		events: make([]ProductionEvent, 0),
	}
}

func (m *memoryRepo) SaveProductionEvent(ctx context.Context, event ProductionEvent, tx ...db.Transaction) error {
	m.events = append(m.events, event)
	return nil
}

func (m *memoryRepo) SaveProduct(ctx context.Context, product Product, tx ...db.Transaction) error {
	m.products = append(m.products, product)
	return nil
}

func (m *memoryRepo) GetProduct(ctx context.Context, sku string, tx ...db.Transaction) (Product, error) {
	for _, p := range m.products {
		if p.Sku == sku {
			return p, nil
		}
	}
	return Product{}, errors.New("product not found")
}

func (m *memoryRepo) GetAllProducts(ctx context.Context, limit int, offset int, tx ...db.Transaction) ([]Product, error) {
	return m.products, nil
}

func (m *memoryRepo) BeginTransaction(ctx context.Context) (db.Transaction, error) {
	return nil, nil
}