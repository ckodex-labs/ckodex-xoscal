.PHONY: all build build-release test test-race coverage proto lint design-lint a11y-lint a11y-live k8s-beta-check site-smoke site-verify fmt security docker clean tidy oscal-check oscal-update dagger-dev dagger-all dagger-test dagger-lint dagger-security dagger-image site site-serve

BINARY := xoscal-server
IMAGE  := xoscal-server
VERSION := $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"

all: build

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./server/cmd/xoscal-server
	go build $(LDFLAGS) -o bin/xoscal-ctl ./server/cmd/xoscal-ctl
	go build $(LDFLAGS) -o bin/xoscal-backup ./server/cmd/xoscal-backup

build-release:
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY) ./server/cmd/xoscal-server
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/xoscal-ctl ./server/cmd/xoscal-ctl
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/xoscal-backup ./server/cmd/xoscal-backup

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
	python3 scripts/a11y-lint.py

design-lint:
	python3 scripts/design-lint.py

a11y-lint:
	python3 scripts/a11y-lint.py

a11y-live:
	python3 scripts/a11y-live.py --root site

k8s-beta-check:
	go test ./server/internal/k8scontract

site-smoke:
	python3 scripts/site-smoke.py --root site

fmt:
	gofmt -w .

security:
	dagger call security-analysis --source=. export --path=security-analysis

docker:
	docker build -t $(IMAGE):$(VERSION) .

clean:
	rm -rf bin/ coverage.out coverage.html gosec-results.sarif

tidy:
	go mod tidy

oscal-check:
	./scripts/reconcile/poll-feeds.sh oscal
	./scripts/reconcile/update-oscal.sh --version latest --check

oscal-update:
	./scripts/reconcile/update-oscal.sh --version "$${OSCAL_VERSION:-latest}"

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
	dagger call snapshot --source=.

dagger-release: dagger-dev
	dagger call release --source=.

site: dagger-dev ## Build the portal site to ./_site (real data; needs engine + network)
	dagger call site --source=. export --path=_site

site-verify: site design-lint a11y-lint ## Build the site, then exercise the exported static surface
	python3 scripts/site-smoke.py --root _site

site-serve: dagger-dev ## Build and serve the portal at http://localhost:8080 (Ctrl-C to stop)
	dagger call serve --source=. up --ports 8080:8080
