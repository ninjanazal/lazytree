.PHONY: build test lint fmt vet clean run help

.DEFAULT_GOAL := build

BINARY := lazytree
CMD := ./cmd/lazytree

build: ## Build the ./lazytree binary
	go build -o $(BINARY) $(CMD)

test: ## Run all tests
	go test ./...

lint: vet fmt-check ## Run vet + fmt-check

vet: ## Run go vet on all packages
	go vet ./...

fmt: ## Auto-format all Go files with gofmt
	gofmt -w .

fmt-check: ## Check formatting without modifying files (used in CI)
	@test -z "$$(gofmt -l .)" || (echo "Run 'make fmt' to fix formatting:" && gofmt -l . && exit 1)

clean: ## Remove the built binary
	rm -f $(BINARY)

run: build ## Build and run against the current directory
	./$(BINARY)

help: ## Show this help
	@printf '\033[1mlazytree — available make targets\033[0m\n\n'
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@printf '\n\033[1mexamples\033[0m\n'
	@printf '  make build\n'
	@printf '  make test\n'
	@printf '  go test ./internal/graph/...                            \033[2m# single package\033[0m\n'
	@printf '  go test ./internal/graph/... -run TestLayoutColumns     \033[2m# single test\033[0m\n'
	@printf '  make run\n'
	@printf '  ./$(BINARY) /path/to/other/repo                         \033[2m# run against a specific repo\033[0m\n'
