GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

DIST=./dist
BINARY_NAME=spot
BINARY_NAME_WINDOWS=$(BINARY_NAME).exe

.PHONY: all restore build-prepare build build-unix build-windows test clean

all: restore test build

restore:
	dep ensure

build-prepare:
	mkdir -p $(DIST)
	
build: build-unix build-windows

build-unix: build-prepare
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(DIST)/$(BINARY_NAME) -v ./cmd/spot
build-windows: build-prepare
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(DIST)/$(BINARY_NAME_WINDOWS) -v ./cmd/spot

test:
	$(GOTEST) -v -cover -race -coverprofile=./coverage.out ./...

clean:
	$(GOCLEAN)
	rm -rf $(DIST)