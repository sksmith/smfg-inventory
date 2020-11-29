package inventory

import (
	"context"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/sksmith/bunnyq"
	"github.com/sksmith/smfg-inventory/db"
)

type MockRepo struct {
	SaveProductionEventFunc           func(ctx context.Context, event *ProductionEvent, tx ...db.Transaction) error
	UpdateReservationFunc             func(ctx context.Context, ID uint64, state ReserveState, qty int64, txs ...db.Transaction) error
	GetProductionEventByRequestIDFunc func(ctx context.Context, requestID string, tx ...db.Transaction) (pe ProductionEvent, err error)
	SaveReservationFunc               func(ctx context.Context, reservation *Reservation, tx ...db.Transaction) error
	GetSkuReservesByStateFunc         func(ctx context.Context, sku string, state ReserveState, limit, offset int, tx ...db.Transaction) ([]Reservation, error)
	SaveProductFunc                   func(ctx context.Context, product Product, tx ...db.Transaction) error
	GetProductFunc                    func(ctx context.Context, sku string, tx ...db.Transaction) (Product, error)
	GetAllProductsFunc                func(ctx context.Context, limit int, offset int, tx ...db.Transaction) ([]Product, error)
	BeginTransactionFunc              func(ctx context.Context) (db.Transaction, error)
	GetReservationByRequestIDFunc     func(ctx context.Context, requestId string, tx ...db.Transaction) (Reservation, error)
}

func (r MockRepo) SaveProductionEvent(ctx context.Context, event *ProductionEvent, tx ...db.Transaction) error {
	return r.SaveProductionEventFunc(ctx, event, tx...)
}

func (r MockRepo) UpdateReservation(ctx context.Context, ID uint64, state ReserveState, qty int64, txs ...db.Transaction) error {
	return r.UpdateReservationFunc(ctx, ID, state, qty, txs...)
}

func (r MockRepo) GetProductionEventByRequestID(ctx context.Context, requestID string, tx ...db.Transaction) (pe ProductionEvent, err error) {
	return r.GetProductionEventByRequestIDFunc(ctx, requestID, tx...)
}

func (r MockRepo) SaveReservation(ctx context.Context, reservation *Reservation, tx ...db.Transaction) error {
	return r.SaveReservationFunc(ctx, reservation, tx...)
}

func (r MockRepo) GetSkuReservationsByState(ctx context.Context, sku string, state ReserveState, limit, offset int, tx ...db.Transaction) ([]Reservation, error) {
	return r.GetSkuReservesByStateFunc(ctx, sku, state, limit, offset, tx...)
}

func (r MockRepo) SaveProduct(ctx context.Context, product Product, tx ...db.Transaction) error {
	return r.SaveProductFunc(ctx, product, tx...)
}

func (r MockRepo) GetProduct(ctx context.Context, sku string, tx ...db.Transaction) (Product, error) {
	return r.GetProductFunc(ctx, sku, tx...)
}

func (r MockRepo) GetAllProducts(ctx context.Context, limit int, offset int, tx ...db.Transaction) ([]Product, error) {
	return r.GetAllProductsFunc(ctx, limit, offset, tx...)
}

func (r MockRepo) BeginTransaction(ctx context.Context) (db.Transaction, error) {
	return r.BeginTransactionFunc(ctx)
}

func (r MockRepo) GetReservationByRequestID(ctx context.Context, requestId string, tx ...db.Transaction) (Reservation, error) {
	return r.GetReservationByRequestIDFunc(ctx, requestId, tx...)
}

func NewMockRepo() MockRepo {
	return MockRepo{
		SaveProductionEventFunc: func(ctx context.Context, event *ProductionEvent, tx ...db.Transaction) error { return nil },
		GetProductionEventByRequestIDFunc: func(ctx context.Context, requestID string, tx ...db.Transaction) (pe ProductionEvent, err error) {
			return ProductionEvent{}, nil
		},
		SaveReservationFunc:       func(ctx context.Context, reservation *Reservation, tx ...db.Transaction) error { return nil },
		GetSkuReservesByStateFunc: func(ctx context.Context, sku string, state ReserveState, limit, offset int, tx ...db.Transaction) ([]Reservation, error) { return nil, nil },
		SaveProductFunc:           func(ctx context.Context, product Product, tx ...db.Transaction) error { return nil },
		GetProductFunc:            func(ctx context.Context, sku string, tx ...db.Transaction) (Product, error) { return Product{}, nil },
		GetAllProductsFunc:        func(ctx context.Context, limit int, offset int, tx ...db.Transaction) ([]Product, error) { return nil, nil },
		BeginTransactionFunc:      func(ctx context.Context) (db.Transaction, error) { return MockTransaction{}, nil },
	}
}

type MockTransaction struct {
}

func (m MockTransaction) Commit(_ context.Context) error {
	return nil
}

func (m MockTransaction) Rollback(_ context.Context) error {
	return nil
}

func (m MockTransaction) Query(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (m MockTransaction) QueryRow(_ context.Context, _ string, _ ...interface{}) pgx.Row {
	return nil
}

func (m MockTransaction) Exec(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
	return nil, nil
}

func (m MockTransaction) Begin(_ context.Context) (pgx.Tx, error) {
	return nil, nil
}

type MockQueue struct {
	PublishFunc func(ctx context.Context, exchange string, body []byte, options ...bunnyq.PublishOption) error
}

func NewMockQueue() MockQueue {
	return MockQueue{
		PublishFunc: func(ctx context.Context, exchange string, body []byte, options ...bunnyq.PublishOption) error {
			return nil
		},
	}
}

func (m MockQueue) Publish(ctx context.Context, exchange string, body []byte, options ...bunnyq.PublishOption) error {
	return m.PublishFunc(ctx, exchange, body, options...)
}
