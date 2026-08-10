.PHONY: all build build-release test test-race coverage proto lint design-lint site-smoke site-verify fmt security docker clean tidy dagger-dev dagger-all dagger-test dagger-lint dagger-security dagger-image site site-serve

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
	python3 scripts/design-lint.py

design-lint:
	python3 scripts/design-lint.py

site-smoke:
	python3 scripts/site-smoke.py --root site

fmt:
	gofmt -w .

security:
	which govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...
	which gosec >/dev/null 2>&1 || go install github.com/securego/gosec/v2/cmd/gosec@latest
	# govulncheck is the blocking dependency gate; gosec emits the SARIF report
	# while excluding generated protobufs and tolerating analyzer-only failures.
	# Limit analyzer concurrency so the report is stable in constrained CI runners.
	gosec -concurrency=2 -fmt sarif -out gosec-results.sarif -exclude-dir=proto -no-fail ./server/...

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

site: dagger-dev ## Build the portal site to ./_site (real data; needs engine + network)
	dagger call site --source=. export --path=_site

site-verify: site design-lint ## Build the site, then exercise the exported static surface
	python3 scripts/site-smoke.py --root _site

site-serve: dagger-dev ## Build and serve the portal at http://localhost:8080 (Ctrl-C to stop)
	dagger call serve --source=. up --ports 8080:8080
