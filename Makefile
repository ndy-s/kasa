.PHONY: run test lint tidy

run:
	cd backend && go run ./cmd/api

test:
	cd backend && go test ./... -race

lint:
	cd backend && golangci-lint run

tidy:
	cd backend && go mod tidy


