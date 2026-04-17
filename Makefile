MOCKERY_VERSION=v3.5.0

install-tools:
	go install github.com/vektra/mockery/v3@$(MOCKERY_VERSION)

mock:
	mockery --config tools/config/mockery.yml