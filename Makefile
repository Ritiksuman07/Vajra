# Vajra — developer tasks
# Usage: make build | make test | make run | make web | make clean

BINARY := vajra
DIST   := dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w

.PHONY: all build build-all test vet fmt tidy run web serve clean install doctor

all: build

build:
	@mkdir -p $(DIST)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY) .

build-all:
	@mkdir -p $(DIST)
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-windows-amd64.exe .
	GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-linux-amd64 .
	GOOS=linux   GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-linux-arm64 .
	GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-darwin-arm64 .
	GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-darwin-amd64 .

test:
	go test ./... -count=1

vet:
	go vet ./...

fmt:
	gofmt -s -w .

tidy:
	go mod tidy

run: build
	./$(DIST)/$(BINARY)

web: build
	./$(DIST)/$(BINARY) web

serve: build
	./$(DIST)/$(BINARY) serve

doctor: build
	./$(DIST)/$(BINARY) doctor

install:
	go install .

clean:
	rm -rf $(DIST) build AppDir *.AppImage *.deb *.rpm
