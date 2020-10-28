# SMFG - Inventory

This service provides several REST endpoints.

* A link to the parent project
* How do you run the project?
* How do you build the project?
* Add a link to the swagger file
* Lookup a nice README to emulate

## Updating the Project 

```shell
git add .
git commit -m "your message"
git tag v0.0.1
git push origin v0.0.1
docker build . -t smfg-inventory:latest
```

```shell
sudo docker build . --tag docker-registry:5000/smfg-inventory:1.0
sudo docker push docker-registry:5000/smfg-inventory:1.0
```

## Running the Project

If you wish to see how this application runs in its complete constellation, see [the parent repo](https://github.com/sksmith/smithmfg).

If you just want to run this specific microservice locally...

### Build the docker container

```shell
docker build . -t smfg-inventory:latest
```

### Run Docker Compose

```shell
docker-compose up
```

## Database Migrations

I'm using the migrate project to manage database migrations.

```shell
migrate create -ext sql -dir db/migrations -seq create_products_table

migrate -database postgres://postgres:postgres@localhost:5432/smfg-db?sslmode=disable -path db/migrations up
```