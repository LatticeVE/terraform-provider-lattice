.PHONY: default build test tidy fmt clean

default: build

build:
	go build -o terraform-provider-lattice

test:
	go test -v ./...

tidy:
	go mod tidy

fmt:
	go fmt ./...

clean:
	rm -f terraform-provider-lattice
