.PHONY: build test lint fmt vet clean

BINARY := lazytree
CMD := ./cmd/lazytree

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./...

lint: vet fmt-check

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Run 'make fmt' to fix formatting:" && gofmt -l . && exit 1)

clean:
	rm -f $(BINARY)

run: build
	./$(BINARY)
