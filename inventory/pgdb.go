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
	tx := d.conn
	if len(txs) > 0 {
		tx = txs[0]
	}
	ct, err := tx.Exec(ctx,`
		UPDATE production_events
           SET sku = $2, quantity = $3, created = $4
         WHERE id = $1;`,
         event.ID, event.Sku, event.Quantity, event.Created)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		ct, err = tx.Exec(ctx,`
			INSERT INTO production_events (sku, quantity, created)
			       VALUES ($1, $2, $3);`,
			event.Sku, event.Quantity, event.Created)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *dbRepo) SaveProduct(ctx context.Context, product Product, txs ...db.Transaction) error {
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
		return err
	}
	if ct.RowsAffected() == 0 {
		_, err := tx.Exec(ctx,`
		INSERT INTO products (sku, upc, name, available, reserved)
                      VALUES ($1, $2, $3, $4, $5);`,
			product.Sku, product.Upc, product.Name, product.Available, product.Reserved)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *dbRepo) GetProduct(ctx context.Context, sku string, txs ...db.Transaction) (Product, error) {
	tx := d.conn
	if len(txs) > 0 {
		tx = txs[0]
	}

	product := Product{}
	err := tx.QueryRow(ctx, `SELECT sku, upc, name, available, reserved FROM products WHERE sku = $1`, sku).
		Scan(&product.Sku, &product.Upc, &product.Name, &product.Available, &product.Reserved)

	if err != nil {
		return product, err
	}

	return product, nil
}

func (d *dbRepo) GetAllProducts(ctx context.Context, limit int, offset int, txs ...db.Transaction) ([]Product, error) {
	tx := d.conn
	if len(txs) > 0 {
		tx = txs[0]
	}

	products := make([]Product, 0)
	rows, err := tx.Query(ctx,
		`SELECT sku, upc, name, available, reserved FROM products ORDER BY sku LIMIT $1 OFFSET $2;`,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		product := Product{}
		err = rows.Scan(&product.Sku, &product.Upc, &product.Name, &product.Available, &product.Reserved)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, nil
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
