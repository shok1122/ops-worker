BINARY     := ops-worker
MODULE     := $(shell go list -m)
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "\
  -X $(MODULE)/version.Version=$(VERSION) \
  -X $(MODULE)/version.Commit=$(COMMIT) \
  -X $(MODULE)/version.BuildDate=$(BUILD_DATE)"

.PHONY: build run test lint vet fmt clean install

build:
	go build $(LDFLAGS) -o $(BINARY) .

run:
	go run $(LDFLAGS) .

test:
	go test ./...

lint:
	go vet ./...

vet: lint

fmt:
	gofmt -w .

clean:
	rm -f $(BINARY)

install: build
	install -m 755 $(BINARY) /usr/local/bin/$(BINARY)
