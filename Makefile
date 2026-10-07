.PHONY: run test vet build up down

run:
	go run ./cmd/api

test:
	go test -race ./...

vet:
	go vet ./...

build:
	CGO_ENABLED=0 go build -o bin/api ./cmd/api

up:
	docker compose up --build

down:
	docker compose down
