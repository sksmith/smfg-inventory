package inventory

import (
	"context"
	"database/sql"
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
	GetReservationByRequestID(ctx context.Context, requestId string, tx ...db.Transaction) (Reservation, error)
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
		if err == pgx.ErrNoRows {
			return product, errors.WithStack(sql.ErrNoRows)
		}
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
			if err == pgx.ErrNoRows {
				return nil, errors.WithStack(sql.ErrNoRows)
			}
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
	err = tx.QueryRow(ctx, `SELECT id, request_id, sku, quantity, created FROM production_events WHERE request_id = $1`, requestID).
		Scan(&pe.ID, &pe.RequestID, &pe.Sku, &pe.Quantity, &pe.Created)

	if err != nil {
		m.Complete(err)
		if err == pgx.ErrNoRows {
			return pe, errors.WithStack(sql.ErrNoRows)
		}
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
		if err == pgx.ErrNoRows {
			return errors.WithStack(sql.ErrNoRows)
		}
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
	insert := `INSERT INTO reservations (request_id, requester, sku, state, reserved_quantity, requested_quantity, created)
                      VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id;`
	err := tx.QueryRow(ctx, insert, r.RequestID, r.Requester, r.Sku, r.State, r.ReservedQuantity, r.RequestedQuantity, r.Created).Scan(&r.ID)
	if err != nil {
		m.Complete(err)
		if err == pgx.ErrNoRows {
			return errors.WithStack(sql.ErrNoRows)
		}
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
	update := `UPDATE reservations SET state = $2, reserved_quantity = $3 WHERE id=$1;`
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
		`SELECT id, request_id, requester, sku, state, reserved_quantity, requested_quantity, created
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
		err = rows.Scan(&r.ID, &r.RequestID, &r.Requester, &r.Sku, &r.State, &r.ReservedQuantity, &r.RequestedQuantity, &r.Created)
		if err != nil {
			m.Complete(err)
			return nil, err
		}
		reservations = append(reservations, r)
	}

	m.Complete(nil)
	return reservations, nil
}

func (d *dbRepo) GetReservationByRequestID(ctx context.Context, requestId string, txs ...db.Transaction) (Reservation, error) {
	m := db.StartMetric("GetReservationByRequestID")
	tx := d.conn
	if len(txs) > 0 {
		tx = txs[0]
	}

	r := Reservation{}
	err := tx.QueryRow(ctx,
		`SELECT id, request_id, requester, sku, state, reserved_quantity, requested_quantity, created
               FROM reservations
              WHERE request_id = $1;`,
		requestId).Scan(&r.ID, &r.RequestID, &r.Requester, &r.Sku, &r.State, &r.ReservedQuantity, &r.RequestedQuantity, &r.Created)
	if err != nil {
		m.Complete(err)
		if err == pgx.ErrNoRows {
			return r, errors.WithStack(sql.ErrNoRows)
		}
		return r, errors.WithStack(err)
	}

	m.Complete(nil)
	return r, nil
}

func (d *dbRepo) BeginTransaction(ctx context.Context) (db.Transaction, error) {
	tx, err := d.conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}
