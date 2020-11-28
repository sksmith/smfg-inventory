CREATE TABLE IF NOT EXISTS production_events(
    id SERIAL PRIMARY KEY,
    request_id VARCHAR(200) NOT NULL,
    sku VARCHAR(50) NOT NULL,
    quantity INTEGER,
    created timestamptz NOT NULL
);

CREATE INDEX pre_sku_idx ON production_events (sku);
CREATE INDEX pre_created_idx ON production_events (created);
CREATE UNIQUE INDEX requestId ON production_events (request_id);

COMMIT;