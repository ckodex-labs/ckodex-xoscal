.PHONY: all build build-release test test-race coverage proto lint fmt security docker clean tidy dagger-dev dagger-all dagger-test dagger-lint dagger-security dagger-image

BINARY := xoscal-server
IMAGE  := xoscal-server
VERSION := $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"

all: build

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./server/cmd/xoscal-server

build-release:
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY) ./server/cmd/xoscal-server

test:
	go test -v ./...

test-race:
	go test -race ./...

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

proto:
	buf generate

lint:
	buf lint
	go vet ./...
	gofmt -d .

fmt:
	gofmt -w .

security:
	which govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...
	which gosec >/dev/null 2>&1 || go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -fmt sarif -out gosec-results.sarif ./...

docker:
	docker build -t $(IMAGE):$(VERSION) .

clean:
	rm -rf bin/ coverage.out coverage.html gosec-results.sarif

tidy:
	go mod tidy

# --- Dagger pipeline targets ---

dagger-dev:
	dagger develop

dagger-all: dagger-dev
	dagger call all --source=.

dagger-test: dagger-dev
	dagger call test --source=.

dagger-lint: dagger-dev
	dagger call lint --source=.

dagger-security: dagger-dev
	dagger call security --source=.

dagger-image: dagger-dev
	dagger call image --source=.

dagger-snapshot: dagger-dev
	dagger call snapshot --source=. --github-token=env:GITHUB_TOKEN

dagger-release: dagger-dev
	dagger call release --source=. --github-token=env:GITHUB_TOKEN
