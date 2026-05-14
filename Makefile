MOCKERY_VERSION=v3.5.0
VERSION = $(shell git describe --tags --always --dirty)

install-tools:
	go install github.com/vektra/mockery/v3@$(MOCKERY_VERSION)

mock:
	mockery --config tools/config/mockery.yml

run:
	@echo "Building with version: $(VERSION)"
	VERSION=$(VERSION) docker compose -f docker-compose.yml -f docker-compose.local.yml up -d --build

test-unit:
	go test ./internal/service

test-functional:
	docker compose -p tuchka-test -f docker-compose.yml -f docker-compose.test.yml down -v

	docker compose -p tuchka-test -f docker-compose.yml -f docker-compose.test.yml up -d --build

	./scripts/wait-for.sh https://127.0.0.1:8443/health

	go test ./tests/functional -v

	docker compose -p tuchka-test down -v

gen-doc:
	swag init -g cmd/tuchka-server/main.go