CREATE TABLE IF NOT EXISTS products(
   sku VARCHAR(50) PRIMARY KEY,
   upc VARCHAR (50) UNIQUE NOT NULL,
   name VARCHAR (100) NOT NULL,
   available INTEGER NOT NULL,
   reserved INTEGER
);

COMMIT;