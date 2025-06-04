include .env
export

.PHONY: run
run:
	cd src/app && \
	go run cmd/api/main.go

.PHONY: devup
devup:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d

.PHONY: devdown
devdown:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml down

ROOT_DIR := $(dir $(realpath $(lastword $(MAKEFILE_LIST))))

.PHONY: mgc
# Example: make mgc filename=create_customer
mgc:
	docker run --rm -v $(ROOT_DIR)migrations:/migrations migrate/migrate -verbose create -ext sql -dir /migrations $(filename)

.PHONY: mgu
mgu:
	docker run --rm --network host -v $(ROOT_DIR)migrations:/migrations migrate/migrate -verbose -path=/migrations/ -database "$(DB_DSN)" up

.PHONY: mgd
mgd:
	docker run --rm --network host -v $(ROOT_DIR)migrations:/migrations migrate/migrate -verbose -path=/migrations/ -database $(DB_DSN) down 1