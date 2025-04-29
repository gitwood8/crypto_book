GO               = go
GOBIN            ?= $(PWD)/bin
PATH             := $(GOBIN):$(PATH)
PROJECT_NAME     = wood_post
M                = $(shell printf "\033[34;1m>>\033[0m")

export GOOSE_MIGRATION_DIR = /app/migrations/
export GOOSE_DBSTRING      = $(STORAGE_MIGRATION_DSN)
export GOOSE_DRIVER        = postgres

export GOBIN
export PATH

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
	GOBIN=$(GOBIN) $(GO) install github.com/air-verse/air@latest
	@echo "Installing goose..."
	GOBIN=$(GOBIN) $(GO) install github.com/pressly/goose/v3/cmd/goose@v3.19.1

.PHONY: db-migrate
db-migrate:
	$(info $(M) Running DB migrations...)
	/app/bin/goose -dir $(GOOSE_MIGRATION_DIR) postgres $(GOOSE_DBSTRING) up


.PHONY: test
test:
	$(info $(M) Running tests...)
	$(GO) test ./...
