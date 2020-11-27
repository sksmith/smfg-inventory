package inventory

import (
	"context"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	"github.com/sksmith/smfg-inventory/db"
)

type Repository interface {
	SaveProductionEvent(ctx context.Context, event *ProductionEvent, tx ...db.Transaction) error
	GetProductionEventByRequestID(ctx context.Context, requestID string, tx ...db.Transaction)  (pe ProductionEvent, err error)
	SaveReservation(ctx context.Context, reservation *Reservation, tx ...db.Transaction) error
	UpdateReservation(ctx context.Context, ID uint64, state ReserveState, qty int64, txs ...db.Transaction) error
	GetSkuReservationsByState(ctx context.Context, sku string, state ReserveState, limit, offset int, tx ...db.Transaction) ([]Reservation, error)
	SaveProduct(ctx context.Context, product Product, tx ...db.Transaction) error
	GetProduct(ctx context.Context, sku string, tx ...db.Transaction) (Product, error)
	GetAllProducts(ctx context.Context, limit int, offset int, tx ...db.Transaction) ([]Product, error)
	BeginTransaction(ctx context.Context) (db.Transaction, error)
}

type dbRepo struct {
	conn db.Conn
}

func NewPostgresRepo(conn db.Conn) Repository {
	return &dbRepo{
		conn: conn,
	}
}

func (d *dbRepo) SaveProduct(ctx context.Context, product Product, txs ...db.Transaction) error {
	m := db.StartMetric("SaveProduct")
	tx := d.conn
	if len(txs) > 0 {
		tx = txs[0]
	}
	ct, err := tx.Exec(ctx,`
		UPDATE products
           SET upc = $2, name = $3, available = $4, reserved = $5
         WHERE sku = $1;`,
		product.Sku, product.Upc, product.Name, product.Available, product.Reserved)
	if err != nil {
		m.Complete(nil)
		return errors.WithStack(err)
	}
	if ct.RowsAffected() == 0 {
		_, err := tx.Exec(ctx,`
		INSERT INTO products (sku, upc, name, available, reserved)
                      VALUES ($1, $2, $3, $4, $5);`,
			product.Sku, product.Upc, product.Name, product.Available, product.Reserved)
		m.Complete(err)
		if err != nil {
			return err
		}
	}
	m.Complete(nil)
	return nil
}

func (d *dbRepo) GetProduct(ctx context.Context, sku string, txs ...db.Transaction) (Product, error) {
	m := db.StartMetric("GetProduct")
	tx := d.conn
	if len(txs) > 0 {
		tx = txs[0]
	}

	product := Product{}
	err := tx.QueryRow(ctx, `SELECT sku, upc, name, available, reserved FROM products WHERE sku = $1`, sku).
		Scan(&product.Sku, &product.Upc, &product.Name, &product.Available, &product.Reserved)

	if err != nil {
		m.Complete(err)
		return product, errors.WithStack(err)
	}

	m.Complete(nil)
	return product, nil
}

func (d *dbRepo) GetAllProducts(ctx context.Context, limit int, offset int, txs ...db.Transaction) ([]Product, error) {
	m := db.StartMetric("GetAllProducts")
	tx := d.conn
	if len(txs) > 0 {
		tx = txs[0]
	}

	products := make([]Product, 0)
	rows, err := tx.Query(ctx,
		`SELECT sku, upc, name, available, reserved FROM products ORDER BY sku LIMIT $1 OFFSET $2;`,
		limit, offset)
	if err != nil {
		m.Complete(err)
		return nil, errors.WithStack(err)
	}
	defer rows.Close()

	for rows.Next() {
		product := Product{}
		err = rows.Scan(&product.Sku, &product.Upc, &product.Name, &product.Available, &product.Reserved)
		if err != nil {
			m.Complete(err)
			return nil, errors.WithStack(err)
		}
		products = append(products, product)
	}

	m.Complete(nil)
	return products, nil
}

func (d *dbRepo) GetProductionEventByRequestID(ctx context.Context, requestID string, txs ...db.Transaction) (pe ProductionEvent, err error) {
	m := db.StartMetric("GetProductionEventByRequestID")
	tx := d.conn
	if len(txs) > 0 {
		tx = txs[0]
	}

	pe = ProductionEvent{}
	err = tx.QueryRow(ctx, `SELECT request_id, sku, quantity, created FROM production_events WHERE request_id = $1`, requestID).
		Scan(&pe.RequestID, &pe.Sku, &pe.Quantity, &pe.Created)

	if err != nil {
		m.Complete(err)
		return pe, errors.WithStack(err)
	}

	m.Complete(nil)
	return pe, nil
}

func (d *dbRepo) SaveProductionEvent(ctx context.Context, event *ProductionEvent, txs ...db.Transaction) error {
	m := db.StartMetric("SaveProductionEvent")
	tx := d.conn
	if len(txs) > 0 {
		tx = txs[0]
	}
	insert := `INSERT INTO production_events (request_id, sku, quantity, created)
			       VALUES ($1, $2, $3, $4) RETURNING id;`

	err := tx.QueryRow(ctx, insert, event.RequestID, event.Sku, event.Quantity, event.Created).Scan(&event.ID)
	if err != nil {
		m.Complete(err)
		return errors.WithStack(err)
	}
	m.Complete(nil)
	return nil
}

func (d *dbRepo) SaveReservation(ctx context.Context, r *Reservation, txs ...db.Transaction) error {
	m := db.StartMetric("SaveReservation")
	tx := d.conn
	if len(txs) > 0 {
		tx = txs[0]
	}
	insert := `INSERT INTO reservations (requester, sku, state, reserved_quantity, requested_quantity)
                      VALUES ($1, $2, $3, $4, $5) RETURNING id;`
	err := tx.QueryRow(ctx, insert, r.Requester, r.Sku, r.State, r.ReservedQuantity, r.RequestedQuantity).Scan(&r.ID)
	m.Complete(err)
	if err != nil {
		return errors.WithStack(err)
	}
	m.Complete(nil)
	return nil
}

func (d *dbRepo) UpdateReservation(ctx context.Context, ID uint64, state ReserveState, qty int64, txs ...db.Transaction) error {
	m := db.StartMetric("UpdateReservation")
	tx := d.conn
	if len(txs) > 0 {
		tx = txs[0]
	}
	update := `UPDATE reservations SET state = $2, reserved_quantity = $3) WHERE id=$1;`
	_, err := tx.Exec(ctx, update, ID, state, qty)
	m.Complete(err)
	if err != nil {
		return errors.WithStack(err)
	}
	m.Complete(nil)
	return nil
}

func (d *dbRepo) GetSkuReservationsByState(ctx context.Context, sku string, state ReserveState, limit, offset int, txs ...db.Transaction) ([]Reservation, error) {
	m := db.StartMetric("GetSkuOpenReserves")
	tx := d.conn
	if len(txs) > 0 {
		tx = txs[0]
	}

	reservations := make([]Reservation, 0)
	rows, err := tx.Query(ctx,
		`SELECT id, requester, sku, state, reserved_quantity, requested_quantity 
               FROM reservations
              WHERE sku = $1 AND state = $2
           ORDER BY created ASC LIMIT $3 OFFSET $4;`,
		sku, state, limit, offset)
	if err != nil {
		m.Complete(err)
		return nil, errors.WithStack(err)
	}
	defer rows.Close()

	for rows.Next() {
		r := Reservation{}
		err = rows.Scan(&r.ID, &r.Requester, &r.Sku, &r.State, &r.ReservedQuantity, &r.RequestedQuantity)
		if err != nil {
			m.Complete(err)
			return nil, err
		}
		reservations = append(reservations, r)
	}

	m.Complete(nil)
	return reservations, nil
}

func (d *dbRepo) BeginTransaction(ctx context.Context) (db.Transaction, error) {
	return d.conn.Begin(ctx)
}

type MockRepo struct {
	SaveProductionEventFunc func(ctx context.Context, event *ProductionEvent, tx ...db.Transaction) error
	UpdateReservationFunc func(ctx context.Context, ID uint64, state ReserveState, qty int64, txs ...db.Transaction) error
	GetProductionEventByRequestIDFunc func(ctx context.Context, requestID string, tx ...db.Transaction) (pe ProductionEvent, err error)
	SaveReservationFunc func(ctx context.Context, reservation *Reservation, tx ...db.Transaction) error
	GetSkuReservesByStateFunc func(ctx context.Context, sku string, state ReserveState, limit, offset int, tx ...db.Transaction) ([]Reservation, error)
	SaveProductFunc func(ctx context.Context, product Product, tx ...db.Transaction) error
	GetProductFunc func(ctx context.Context, sku string, tx ...db.Transaction) (Product, error)
	GetAllProductsFunc func(ctx context.Context, limit int, offset int, tx ...db.Transaction) ([]Product, error)
	BeginTransactionFunc func(ctx context.Context) (db.Transaction, error)
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

func NewMockRepo() MockRepo {
	return MockRepo{
		SaveProductionEventFunc:   func(ctx context.Context, event *ProductionEvent, tx ...db.Transaction) error {return nil},
		GetProductionEventByRequestIDFunc: func(ctx context.Context, requestID string, tx ...db.Transaction) (pe ProductionEvent, err error) {
			return ProductionEvent{}, nil
		},
		SaveReservationFunc:       func(ctx context.Context, reservation *Reservation, tx ...db.Transaction) error {return nil},
		GetSkuReservesByStateFunc: func(ctx context.Context, sku string, state ReserveState, limit, offset int, tx ...db.Transaction) ([]Reservation, error) {return nil, nil},
		SaveProductFunc:           func(ctx context.Context, product Product, tx ...db.Transaction) error {return nil},
		GetProductFunc:            func(ctx context.Context, sku string, tx ...db.Transaction) (Product, error) {return Product{}, nil},
		GetAllProductsFunc:        func(ctx context.Context, limit int, offset int, tx ...db.Transaction) ([]Product, error) {return nil, nil},
		BeginTransactionFunc:      func(ctx context.Context) (db.Transaction, error) { return MockTransaction{}, nil},
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
