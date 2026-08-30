BINARY  := modhaven
PKG     := ./cmd/modhaven
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

# Platforms the editor is released for. Space Haven runs on all three, and a Go
# binary needs no runtime installed on any of them.
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build
build: ## Build the editor into bin/
	@mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) $(PKG)

.PHONY: install
install: ## Install the editor into GOPATH/bin
	go install -trimpath -ldflags "$(LDFLAGS)" $(PKG)

.PHONY: run
run: ## Build and start the terminal interface
	go run $(PKG)

.PHONY: test
test: ## Run the test suite
	go test ./...

.PHONY: cover
cover: ## Run the tests and open a coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "wrote coverage.html"

.PHONY: race
race: ## Run the test suite under the race detector
	go test -race ./...

.PHONY: check
check: fmt vet test ## Format, vet and test

.PHONY: fmt
fmt: ## Format the source
	gofmt -l -w .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: release
release: ## Cross-compile a binary for every supported platform
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		out=dist/$(BINARY)_$${os}_$${arch}; \
		if [ "$$os" = "windows" ]; then out=$$out.exe; fi; \
		echo "building $$out"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
			go build -trimpath -ldflags "$(LDFLAGS)" -o $$out $(PKG) || exit 1; \
	done

.PHONY: clean
clean: ## Remove build output
	rm -rf bin dist coverage.out coverage.html

.PHONY: help
help: ## List the available targets
	@grep -E '^[a-z-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-10s %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
