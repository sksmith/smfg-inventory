CREATE TABLE IF NOT EXISTS reservations(
    id SERIAL PRIMARY KEY,
    request_id VARCHAR(200) NOT NULL,
    requester VARCHAR(50) NOT NULL,
    sku VARCHAR(50) NOT NULL,
    state VARCHAR(50) NOT NULL,
    reserved_quantity INTEGER,
    requested_quantity INTEGER,
    created timestamptz NOT NULL
);

CREATE INDEX res_lookup_idx ON reservations (sku, state);
CREATE INDEX res_requester_idx ON reservations (requester);

COMMIT;