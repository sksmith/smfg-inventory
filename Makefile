build:
    echo "Building the binary"
	go build -o bin/smfg-inventory .

test:
    go test -v ./...

run:
    echo "executing the application"
	go run .

publish:
    VER=$(shell echo git describe --tag)
    echo VER