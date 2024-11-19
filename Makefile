NAME=testkit
REPO=github.com/bdpiprava/${NAME}

BUILD_DIR=build

## Run tests
tests:
	@go install github.com/mfridman/tparse@latest
	@go test -race=1 ./... -json | tparse -follow -pass

## Remove build and vendor directory
clean:
	@rm -rf build/
	@rm -rf vendor/

## Build the binary
build:
	@go build -o build/ -v ./...

## Install dependencies
deps:
	@go mod tidy
	@go mod vendor
	@go get .

## Install the binary
install:
	@go install ${REPO}