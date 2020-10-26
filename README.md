# SMFG - Inventory

This service provides several REST endpoints.

* A link to the parent project
* How do you run the project?
* How do you build the project?
* Add a link to the swagger file
* Lookup a nice README to emulate

```shell
sudo docker build . --tag docker-registry:5000/smfg-inventory:1.0
sudo docker push docker-registry:5000/smfg-inventory:1.0
```

## Database Migrations

I'm using the migrate project to manage database migrations.

```shell
migrate create -ext sql -dir db/migrations -seq create_products_table

migrate -database postgres://postgres:postgres@localhost:5432/smfg-db?sslmode=disable -path db/migrations up
```