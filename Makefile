include .envrc

BIN=coachgpt

.PHONY: dev run build migrate-up migrate-down sqlc gen tidy fmt worker manual-sync test smoke lint lint-fix

dev:
	@echo "Starting API server and worker..."
	@go run ./cmd/api & go run ./cmd/worker & wait
run:
	go run ./cmd/api
worker:
	go run ./cmd/worker
test:
	go test ./... -v
smoke:
	go test -v ./smoke_test.go -run TestSmokeTest
lint:
	golangci-lint run
lint-fix:
	golangci-lint run --fix
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
