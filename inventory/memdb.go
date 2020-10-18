package inventory

import (
	"context"
	"errors"
	"github.com/sksmith/smfg-inventory/db"
	"sync"
)

type memoryRepo struct {
	products *sync.Map
	events *sync.Map
	prodEvtId uint64
}

func NewMemoryRepo() Repository {
	products := &sync.Map{}
	products.Store("SSPROCK01", Product{
			Sku:       "SSPROCK01",
			Upc:       "10235668",
			Name:      "Small Basic Sprocket",
			Available: 1,
			Reserved:  0,
		},
	)

	products.Store("SSPROCK02", Product{
		Sku:       "SSPROCK02",
		Upc:       "1255506827",
		Name:      "Small Advanced Sprocket",
		Available: 3,
		Reserved:  0,
	},
	)

	products.Store("LSPROCK01", Product{
		Sku:       "LSPROCK01",
		Upc:       "4670235668",
		Name:      "Large Simple Sprocket",
		Available: 2,
		Reserved:  0,
	},
	)

	return &memoryRepo{
		products:  products,
		events:    &sync.Map{},
		prodEvtId: 0,
	}
}

func (m *memoryRepo) getEventId() uint64 {
	m.prodEvtId++
	return m.prodEvtId
}

func (m *memoryRepo) SaveProductionEvent(_ context.Context, event ProductionEvent, _ ...db.Transaction) error {
	if event.ID == 0 {
		event.ID = m.getEventId()
	}
	m.events.Store(event.ID, event)
	return nil
}

func (m *memoryRepo) SaveProduct(_ context.Context, product Product, _ ...db.Transaction) error {
	m.products.Store(product.Sku, product)
	return nil
}

func (m *memoryRepo) GetProduct(_ context.Context, sku string, _ ...db.Transaction) (Product, error) {
	product, ok := m.products.Load(sku)
	if !ok {
		return Product{}, errors.New("product not found")
	}
	return product.(Product), nil
}

// GetAllProducts Note: The in memory db just uses a simple map for storage. There is no guarantee of proper ordering
// with a map so pagination will not function as expected.
func (m *memoryRepo) GetAllProducts(_ context.Context, limit int, offset int, _ ...db.Transaction) ([]Product, error) {
	products := make([]Product, 0)
	i := 0
	j := 0
	m.products.Range(func(k, v interface{}) bool {
		product := v.(Product)

		if offset <= i {
			if limit > 0 && limit <= j {
				return false
			}
			products = append(products, product)
			j++
			if len(products) < j {
				return false
			}
		}
		i++
		return true
	})

	return products, nil
}

func (m *memoryRepo) BeginTransaction(_ context.Context) (db.Transaction, error) {
	return FalseTx{}, nil
}

type FalseTx struct{}

func (FalseTx) Commit(_ context.Context) error {
	return nil
}

func (FalseTx) Rollback(_ context.Context) error {
	return nil
}

func (FalseTx) RollbackUnlessCommitted(_ context.Context) error {
	return nil
}
