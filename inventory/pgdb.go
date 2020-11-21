package inventory

import (
	"context"
	"github.com/sksmith/smfg-inventory/db"
)

type dbRepo struct {
	conn db.Conn
}

func NewPostgresRepo(conn db.Conn) Repository {
	return &dbRepo{
		conn: conn,
	}
}

func (d *dbRepo) SaveProductionEvent(ctx context.Context, event ProductionEvent, txs ...db.Transaction) error {
	m := db.StartMetric("SaveProductionEvent")
	tx := d.conn
	if len(txs) > 0 {
		tx = txs[0]
	}
	_, err := tx.Exec(ctx,`
			INSERT INTO production_events (sku, quantity, created)
			       VALUES ($1, $2, $3);`,
		event.Sku, event.Quantity, event.Created)
	if err != nil {
		m.Complete(err)
		return err
	}
	m.Complete(nil)
	return nil
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
		return err
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
		return product, err
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
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		product := Product{}
		err = rows.Scan(&product.Sku, &product.Upc, &product.Name, &product.Available, &product.Reserved)
		if err != nil {
			m.Complete(err)
			return nil, err
		}
		products = append(products, product)
	}

	m.Complete(nil)
	return products, nil
}

func (d *dbRepo) SaveReservation(ctx context.Context, r *Reservation, txs ...db.Transaction) error {
	m := db.StartMetric("SaveReservation")
	tx := d.conn
	if len(txs) > 0 {
		tx = txs[0]
	}
	ct, err := tx.Exec(ctx,`
		UPDATE reservations
           SET requester = $2, sku = $3, state = $4, reserved_quantity = $5, requested_quantity
         WHERE id = $1;`,
		r.ID, r.Requester, r.Sku, r.State, r.ReservedQuantity, r.RequestedQuantity)
	if err != nil {
		m.Complete(nil)
		return err
	}
	if ct.RowsAffected() == 0 {
		insert := ` INSERT INTO reservations (requester, sku, state, reserved_quantity, requested_quantity)
                      VALUES ($1, $2, $3, $4, $5) RETURNING id;`
		err = tx.QueryRow(ctx, insert, r.Requester, r.Sku, r.State, r.ReservedQuantity, r.RequestedQuantity).Scan(&r.ID)
		m.Complete(err)
		if err != nil {
			return err
		}
	}
	m.Complete(nil)
	return nil
}

func (d *dbRepo) GetSkuReservesByState(ctx context.Context, sku string, state ReserveState, limit, offset int, txs ...db.Transaction) ([]Reservation, error) {
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
		return nil, err
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

type PostgresTx struct{}

func (PostgresTx) Commit(_ context.Context) error {
	return nil
}

func (PostgresTx) Rollback(_ context.Context) error {
	return nil
}

func (PostgresTx) RollbackUnlessCommitted(_ context.Context) error {
	return nil
}
