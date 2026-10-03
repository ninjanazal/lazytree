.PHONY: build test lint fmt vet clean run demo perf help

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

fmt-check: ## Check formatting without modifying files (used in CI, see .github/workflows/ci.yml)
	@test -z "$$(gofmt -l .)" || (echo "Run 'make fmt' to fix formatting:" && gofmt -l . && exit 1)

clean: ## Remove the built binary
	rm -f $(BINARY)

run: build ## Build and run against the current directory
	./$(BINARY)

demo: ## Build and run against a generated multi-branch test repo
	@./scripts/demo-repo.sh --run

perf: ## Measure load/scroll/search on a synthetic 100k-commit repo
	@dir=$$(./scripts/big-repo.sh) && LAZYTREE_BIGREPO=$$dir go test ./internal/ui -run LargeRepo -v; rm -rf $$dir

help: ## Show this help
	@printf '\033[1mlazytree — available make targets\033[0m\n\n'
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@printf '\n\033[1mexamples\033[0m\n'
	@printf '  make build\n'
	@printf '  make test\n'
	@printf '  go test ./internal/graph/...                            \033[2m# single package\033[0m\n'
	@printf '  go test ./internal/graph/... -run TestLayoutColumns     \033[2m# single test\033[0m\n'
	@printf '  make run\n'
	@printf '  make demo                                                \033[2m# multi-branch test repo\033[0m\n'
	@printf '  ./$(BINARY) /path/to/other/repo                         \033[2m# run against a specific repo\033[0m\n'
