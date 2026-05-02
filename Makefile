.PHONY: run build test test-v cover vet fmt lint lint-install tidy vulncheck check

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test ./...

test-v:
	go test -v ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

cover-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "open coverage.html"

vet:
	go vet ./...

fmt:
	gofmt -w .

lint:
	golangci-lint run ./...

lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

tidy:
	go mod tidy

vulncheck:
	@which govulncheck > /dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

# check ejecuta el set mínimo que debería pasar antes de un commit/PR.
check: fmt vet test
