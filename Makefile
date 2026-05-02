.PHONY: run test fmt lint lint-install tidy

run:
	go run cmd/api/main.go

test:
	go test ./...

test-v:
	go test -v ./...

fmt:
	gofmt -w .

lint:
	golangci-lint run ./...

lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

tidy:
	go mod tidy
