CREATE TABLE IF NOT EXISTS production_events(
    id SERIAL PRIMARY KEY,
    sku VARCHAR(50) NOT NULL,
    quantity INTEGER,
    created timestamptz NOT NULL
);

CREATE INDEX pre_sku_idx ON production_events (sku);
CREATE INDEX pre_created_idx ON production_events (created);

COMMIT;