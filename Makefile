# Set defaults
GO               = go
GOBIN            ?= $(PWD)/bin
PATH             := $(GOBIN):$(PATH)
PROJECT_NAME     = wood_post
M                = $(shell printf "\033[34;1m>>\033[0m")

export GOOSE_MIGRATION_DIR = ./migrations/
export GOOSE_DBSTRING      = $(STORAGE_MIGRATION_DSN)
export GOOSE_DRIVER        = postgres

.PHONY: build-service
build-service:
	$(info $(M) building service...)
	$(GO) build -o $(GOBIN)/$(PROJECT_NAME) ./cmd/service/*.go

watch:
	@go install github.com/air-verse/air@latest
	air -c .air.toml

.PHONY: install-tools
install-tools:
	@echo "Installing air..."
	@go install github.com/air-verse/air@latest

	@echo "Installing goose..."
	@go install github.com/pressly/goose/v3/cmd/goose@v3.19.1


.PHONY: db-migrate
db-migrate:
	$(info $(M) Running DB migrations...)
	@/app/bin/goose -dir $(GOOSE_MIGRATION_DIR) postgres "$(GOOSE_DBSTRING)" up

# Тесты
.PHONY: test
test:
	$(info $(M) Running tests...)
	$(GO) test ./...
