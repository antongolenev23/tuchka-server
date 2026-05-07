MOCKERY_VERSION=v3.5.0
VERSION = $(shell git describe --tags --always --dirty)

install-tools:
	go install github.com/vektra/mockery/v3@$(MOCKERY_VERSION)

mock:
	mockery --config tools/config/mockery.yml

run:
	@echo "Building with version: $(VERSION)"
	VERSION=$(VERSION) docker compose -f docker-compose.yml -f docker-compose.local.yml up --build

test:
	go test ./internal/service
