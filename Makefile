include .env
export

.PHONY: run
run:
	go run cmd/api/main.go

.PHONY: devup
devup:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d

.PHONY: devdown
devdown:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml down