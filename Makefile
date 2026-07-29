.PHONY: run test lint tidy migrate

run:
	cd backend && go run ./cmd/api

test:
	cd backend && go test ./... -race

lint:
	cd backend && golangci-lint run

tidy:
	cd backend && go mod tidy

migrate:
	cd backend && set -a && . ./.env && set +a && goose -dir db/migrations postgres "$$DATABASE_URL" up
