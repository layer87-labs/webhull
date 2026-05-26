BINARY_NAME  := webhull
BUILD_DIR    := build/bin
CONFIG       := deploy/config.yaml
PAGES        := examples/multi-page/pages.yaml

# Parameters — must stay in sync with the pipeline configuration
STATICCHECK_CHECKS ?= all,-ST1000,-U1000
TEST_FLAGS         ?= -v -buildvcs
AUDIT_BINARY       ?= $(BUILD_DIR)/$(BINARY_NAME)

# All handwritten Go files — excludes templ-generated code
GO_FILES := $(shell find . -name "*.go" -not -name "*_templ.go" -not -path "./vendor/*")

.PHONY: all build run dev clean \
        generate \
        fmt fmt-check \
        vet lint shadow audit \
        test test-cover \
        check \
        tidy \
        compress container help

# ── Code generation ──────────────────────────────────────────────────────────

## generate: regenerate templ templates
generate:
	@command -v templ > /dev/null 2>&1 || go install github.com/a-h/templ/cmd/templ@latest
	@templ generate

# ── Formatting ───────────────────────────────────────────────────────────────

## fmt: format all handwritten Go files with gofumpt (superset of gofmt)
fmt:
	@go tool gofumpt -w $(GO_FILES)

## fmt-check: fail if any file is not properly formatted (used in CI / check)
fmt-check:
	@out=$$(go tool gofumpt -l $(GO_FILES)); \
	if [ -n "$$out" ]; then \
		echo "$$out"; \
		echo "→ run 'make fmt' to fix"; \
		exit 1; \
	fi

# ── Static analysis ──────────────────────────────────────────────────────────

## vet: run go vet across all packages
vet:
	@go vet ./...

## lint: staticcheck (all checks except ST1000 and U1000)
lint:
	@go tool staticcheck -checks=$(STATICCHECK_CHECKS) ./...

## shadow: detect variable shadowing in handwritten code
shadow:
	@go tool shadow ./... 2>&1 \
		| grep -v "_templ.go" \
		| grep . \
		&& exit 1 || true

## audit: go mod verify + vulnerability scan (binary mode when binary exists)
audit:
	@go mod verify
	@if [ -f $(AUDIT_BINARY) ]; then \
		go tool govulncheck -mode binary $(AUDIT_BINARY); \
	else \
		go tool govulncheck ./...; \
	fi

# ── Tests ────────────────────────────────────────────────────────────────────

## test: run all tests with race detector
test:
	@go test ./... -race -count=1 $(TEST_FLAGS)

## test-cover: run tests and open HTML coverage report
test-cover:
	@mkdir -p build/coverage
	@go test ./... -coverprofile=build/coverage/coverage.out
	@go tool cover -html=build/coverage/coverage.out -o build/coverage/coverage.html
	@echo "→ build/coverage/coverage.html"

# ── Mandatory gate ───────────────────────────────────────────────────────────

## check: full quality gate — fmt-check, vet, lint, shadow, test, audit
##        run this before every commit; CI must pass all steps
check: fmt-check vet lint shadow test audit

# ── Build ────────────────────────────────────────────────────────────────────

## build: generate templates then compile the binary
build: generate
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/webhull

## ci/build: cross-compile check used in CI (no templ generate — committed)
ci/build:
	@CGO_ENABLED=0 go build ./cmd/webhull

## build/single: build a versioned binary for release (CI use)
## Usage: make build/single SUFFIX=linux-amd64 VERSION=1.2.3
build/single:
	@mkdir -p build/package
	@CGO_ENABLED=0 go build \
		-ldflags="-s -w -X github.com/layer87-labs/webhull/cmd/webhull/cmd.Version=$(VERSION)" \
		-o "build/package/$(BINARY_NAME)_$(VERSION)_$(SUFFIX)$(EXT)" \
		./cmd/webhull

## run: build and start the server
run: build
	@$(BUILD_DIR)/$(BINARY_NAME) -config $(CONFIG) -pages $(PAGES)

## dev: hot-reload development server via air
dev:
	@command -v air > /dev/null 2>&1 || go install github.com/air-verse/air@latest
	@air -c .air.toml

# ── Maintenance ──────────────────────────────────────────────────────────────

## tidy: tidy and verify the module graph
tidy:
	@go mod tidy
	@go mod verify

## clean: remove build artifacts
clean:
	@rm -rf $(BUILD_DIR) build/coverage

## compress: compress the binary with UPX (requires upx installed)
compress:
	@command -v upx > /dev/null 2>&1 || (echo "upx not installed" && exit 1)
	@upx --best --lzma $(BUILD_DIR)/$(BINARY_NAME)

## container: build the container image
container: build
	@docker build -f deploy/Containerfile -t $(BINARY_NAME):latest .

# ── Help ─────────────────────────────────────────────────────────────────────

## help: list all available targets
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /'
