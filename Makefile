include .envrc

BIN=coachgpt

.PHONY: dev run build migrate-up migrate-down sqlc gen tidy fmt

dev:
	go run ./cmd/api
run:
	go run ./cmd/api
build:
	go build -o $(BIN) ./cmd/api
migrate-up:
	goose -dir ./internal/migrations postgres "$$DATABASE_URL" up
migrate-down:
	goose -dir ./internal/migrations postgres "$$DATABASE_URL" down
sqlc:
	sqlc generate

gen: tidy fmt sqlc
tidy:
	go mod tidy
fmt:
	go fmt ./...
