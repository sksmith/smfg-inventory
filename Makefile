VER := $(shell git describe --tag)
SHA1 := $(shell git rev-parse HEAD)
NOW := $(shell date +'%Y-%m-%d_%T') 

build:
	@echo Building the binary
	go build -ldflags "-X main.AppVersion=$(VER) -X main.Sha1Version=$(SHA1) -X main.BuildTime=$(NOW)" -o bin/smfg-inventory .

test:
	go test -v ./...

run:
	echo "executing the application"
	go run .

publish:
	@echo $(VER)

docker:
	docker buildx build --platform linux/arm64 -t docker.seanksmith.me/smfg-inventory:v1.0.0 --push .