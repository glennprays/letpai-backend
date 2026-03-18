RENAME_MODULE_FROM=github.com/glennprays/golang-clean-arch-starter
RENAME_MODULE_TO=

# detect OS
UNAME_S := $(shell uname -s)

rename:
	@if [ -z "$(RENAME_MODULE_TO)" ]; then \
		echo "Please provide RENAME_MODULE_TO, e.g., make rename RENAME_MODULE_TO=github.com/yourname/yourproject"; \
		exit 1; \
	fi
	@if [ "$(UNAME_S)" = "Darwin" ]; then \
		sed -i '' "s|$(RENAME_MODULE_FROM)|$(RENAME_MODULE_TO)|g" go.mod; \
		find . -type f -name '*.go' -exec sed -i '' "s|$(RENAME_MODULE_FROM)|$(RENAME_MODULE_TO)|g" {} +; \
	else \
		sed -i "s|$(RENAME_MODULE_FROM)|$(RENAME_MODULE_TO)|g" go.mod; \
		find . -type f -name '*.go' -exec sed -i "s|$(RENAME_MODULE_FROM)|$(RENAME_MODULE_TO)|g" {} +; \
	fi
	go mod tidy

run-dev:
	@echo "Starting all Docker containers for development mode..."
	@sh docker-dev up -d

stop-dev:
	@echo "Stopping all Docker containers..."
	@sh docker-dev down

swagger:
	@echo "Restarting Swagger UI..."
	@sh docker-dev restart swagger-ui

generate:
	@echo "Generating Wire dependencies..."
	@wire gen ./internal/infrastructure/...

run:
	@go run cmd/api/main.go
