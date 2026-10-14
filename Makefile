.PHONY: build test lint fmt vet clean run demo perf cross docker-test docker-matrix docker-perf screenshots vault vault-preview dist release release-dry check help

.DEFAULT_GOAL := build

BINARY := lazytree
CMD := ./cmd/lazytree

##@ Build & run
build: ## Build the ./lazytree binary
	go build -o $(BINARY) $(CMD)


##@ Quality
test: ## Run all tests
	go test ./...

lint: vet fmt-check ## Run vet + fmt-check

vet: ## Run go vet on all packages
	go vet ./...

fmt: ## Auto-format all Go files with gofmt
	gofmt -w .

fmt-check: ## Check formatting without modifying files
	@test -z "$$(gofmt -l .)" || (echo "Run 'make fmt' to fix formatting:" && gofmt -l . && exit 1)


##@ Run & clean
clean: ## Remove the built binary
	rm -f $(BINARY)

run: build ## Build and run against the current directory
	./$(BINARY)

demo: ## Build and run against a generated multi-branch test repo
	@./scripts/demo-repo.sh --run

perf: ## Measure load/scroll/search on a synthetic 100k-commit repo
	@dir=$$(./scripts/big-repo.sh) && LAZYTREE_BIGREPO=$$dir go test ./internal/ui -run LargeRepo -v; rm -rf $$dir


##@ Verification
DOCKER_BASES := golang:1.26-bookworm golang:1.26-trixie golang:1.26-alpine
GORELEASER := go run github.com/goreleaser/goreleaser/v2@latest

cross: ## Compile-check every release target and Windows (no tests run)
	@for t in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do \
		echo "build $$t"; GOOS=$${t%/*} GOARCH=$${t#*/} CGO_ENABLED=0 go build -o /dev/null $(CMD) || exit 1; \
	done
	@echo "vet windows/amd64" && GOOS=windows GOARCH=amd64 go vet ./...

check: lint test cross ## Everything that runs without Docker: lint, tests, cross-compile

docker-test: ## Run lint + tests in a clean Linux container (Debian)
	docker build -q -f docker/Dockerfile -t lazytree-test . >/dev/null
	docker run --rm lazytree-test

docker-matrix: ## Run lint + tests on Debian bookworm, Debian trixie and Alpine (different git versions)
	@for base in $(DOCKER_BASES); do \
		echo "\n######## $$base"; \
		docker build -q -f docker/Dockerfile --build-arg BASE=$$base -t lazytree-test:$$(echo $$base | tr ':/' '--') . >/dev/null || exit 1; \
		docker run --rm lazytree-test:$$(echo $$base | tr ':/' '--') || exit 1; \
	done

docker-perf: ## Run the 100k-commit performance check in a container
	docker build -q -f docker/Dockerfile -t lazytree-test . >/dev/null
	docker run --rm lazytree-test make perf


##@ Docs & vault
screenshots: ## Regenerate docs/img (README screenshots + demo GIF) with vhs in Docker
	@./scripts/screenshots.sh

vault: ## Regenerate the Obsidian vault pages (Excalidraw) from scripts/vault
	python3 scripts/vault/build.py

vault-preview: ## Rebuild the vault and render PNG previews into ./vault-preview (needs Docker)
	python3 scripts/vault/build.py --preview $(CURDIR)/vault-preview


##@ Release
dist: ## Build release archives into ./dist (local only, never publishes anything)
	$(GORELEASER) release --snapshot --clean --skip=publish

release: ## Prepare a release locally: checks, tag, archives, release text. Usage: make release VERSION=v0.1.0 (never pushes)
	@./scripts/release.sh "$(VERSION)"

release-dry: ## Preview a release (no tag created). Usage: make release-dry VERSION=v0.1.0
	@./scripts/release.sh "$(VERSION)" --dry-run


##@ Help
help: ## Show this help
	@awk -v c="$(if $(NO_COLOR),,1)" 'BEGIN { FS = ":.*## "; \
		if (c) { b="\033[1m"; g="\033[1;35m"; t="\033[36m"; d="\033[2m"; r="\033[0m" } \
		printf "%slazytree — make targets%s\n", b, r } \
		/^##@ / { printf "\n%s▌ %s%s\n", g, substr($$0, 5), r; next } \
		/^[a-zA-Z_-]+:.*## / { printf "  %s%-16s%s %s\n", t, $$1, r, $$2 }' $(MAKEFILE_LIST)
	@printf '\n\033[1;35m▌ Examples\033[0m\n'
	@printf '  \033[36mmake build\033[0m\n'
	@printf '  \033[36mmake test\033[0m\n'
	@printf '  \033[36mgo test ./internal/graph/...\033[0m                           \033[2m# single package\033[0m\n'
	@printf '  \033[36mgo test ./internal/graph/... -run TestLayoutColumns\033[0m    \033[2m# single test\033[0m\n'
	@printf '  \033[36mmake demo\033[0m                                              \033[2m# multi-branch test repo\033[0m\n'
	@printf '  \033[36m./$(BINARY) /path/to/repo\033[0m                               \033[2m# run against a specific repo\033[0m\n'
