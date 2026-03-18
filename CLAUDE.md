# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go Clean Architecture boilerplate template using Fiber framework with Wire dependency injection. Requires Go 1.25+, Docker, and Make.

## Common Commands

```bash
# Run the API server
make run

# Run development services (PostgreSQL, Swagger UI)
make run-dev

# Stop development services
make stop-dev

# Refresh Swagger UI after updating docs
make swagger

# Rename module for new project
make rename RENAME_MODULE_TO=github.com/yourname/yourproject

# Regenerate Wire dependencies (after modifying wire.go)
go generate ./internal/infrastructure/...
```

## Architecture

### Clean Architecture Layers

```
cmd/api/main.go           # Entry point, Fiber server
internal/
├── config/               # Configuration loading
├── domain/               # Core entities & errors (innermost)
├── usecase/              # Application business rules
├── service/              # Reusable business logic
├── repository/           # Data access layer
├── handler/              # HTTP handlers (Fiber)
├── middleware/           # HTTP middleware
├── router/               # Route definitions
├── httperror/            # HTTP error conversion
├── infrastructure/       # Wire DI, external configs
├── utils/                # Helper utilities
└── worker/               # Background jobs
```

### Dependency Rule

Dependencies point inward:
- `domain` → no dependencies on other layers
- `usecase/service` → can import `domain`
- `repository` → can import `domain`
- `handler` → can import `usecase`, `service`, `domain`

### Key Components

**Domain** (`internal/domain/`):
- Error types: `ErrBadRequest`, `ErrNotFound`, `ErrUnauthorized`, `ErrForbidden`, `ErrConflict`, `ErrInternalFailure`
- Use `domain.NewError(serviceErr, appErr)` for domain errors

**Infrastructure** (`internal/infrastructure/`):
- `app.go` - App struct holding dependencies
- `wire.go` / `wire_gen.go` - Wire DI configuration

**Error Handling**:
- `httperror.FromError()` converts domain errors to HTTP status codes

### API Endpoints

- `GET /health` - Health check

### Development Environment

Docker Compose (`misc/develop/docker-compose.yml`):
- PostgreSQL 16.3 on port 5432
- Swagger UI on port 8080

Environment: `.env` (copy from `.env.example`). App runs on port 3000.
