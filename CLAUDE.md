# CLAUDE.md - Letpai Backend Development

**This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.**

---

## Project Overview

Letpai is a bill splitting application with WhatsApp integration. Backend built with Go + Fiber + PostgreSQL using Hexagonal Architecture (DDD) principles.

### Tech Stack
- **Framework**: Fiber v2 (high-performance HTTP framework)
- **Architecture**: Hexagonal Architecture (Domain-Driven Design)
- **Database**: PostgreSQL (via Docker or Railway)
- **Caching**: Redis (for rate limiting)
- **Image Storage**: S3 / Cloudinary
- **WhatsApp Integration**: External gateway via `github.com/glennprays/whatsapp-gateway-sdk-go`
- **Documentation**: Swagger/OpenAPI 3.1.0 (`docs/swagger.yaml`)

---

## Development Context

### IMPORTANT: Read Brief Files First

Before starting any task, ALWAYS read these brief files to understand the context:

1. **API Specification**: `~/.openclaw/workspace/letpai-api-context.md`
   - Complete API endpoints with request/response schemas
   - Error handling and rate limiting rules
   - All features for MVP v1.0

2. **Features Overview**: `~/.openclaw/workspace/letpai-features-concise.md`
   - MVP features (concise version)
   - Core flows (auth, contacts, sessions, payments)
   - UI/UX requirements

3. **Database Schema**: `~/.openclaw/workspace/letpai-database-schema.md`
   - Complete database schema with all tables
   - Indexes, relationships, and constraints
   - Common queries and optimization tips

4. **Dev Guide**: `~/.openclaw/workspace/letpai-backend-dev-guide.md`
   - Detailed implementation phases
   - Step-by-step workflow
   - Architecture decisions

5. **API Documentation**: `docs/swagger.yaml`
   - OpenAPI 3.1.0 specification
   - All endpoints, schemas, parameters
   - Use this for handler implementation reference

### Context7 Integration

**ALWAYS use Context7 MCP server** for library/API documentation:
- Go libraries (Fiber, JWT libraries, PostgreSQL drivers)
- Up-to-date code examples
- No hallucinations (only real APIs)

Context7 is already configured and available via MCP. Auto-invoke it when you need:
- Code generation
- Setup or configuration steps
- Library/API documentation

---

## Architecture Overview

### Layer Structure (Hexagonal Architecture)

```
cmd/api/main.go           # Entry point
domain/                   # Domain layer (root level - DDD)
├── entity/              # Domain entities
├── ports/               # Repository interfaces
└── valueobject/         # Value objects
internal/
├── config/              # Configuration loading
├── usecase/             # Application business rules
├── service/             # Reusable services (JWT, WhatsApp, etc.)
├── repository/          # Data access implementations
├── handler/             # HTTP handlers
├── middleware/          # HTTP middleware
├── router/              # Route definitions
├── httperror/           # HTTP error conversion
├── infrastructure/      # Wire DI, DB, Redis
├── params/              # Request/Response DTOs
│   ├── request/
│   └── response/
└── utils/               # Helper utilities
pkg/
└── logger/              # Shared logger
```

### Dependency Rules (Hexagonal Architecture)

- `domain/` → no dependencies (pure DDD, core business logic)
- `domain/ports/` → interfaces only (contracts for repositories)
- `usecase/` → can import `domain` and `domain/ports`
- `service/` → can import `domain`
- `repository/` → implements `domain/ports`, imports `domain`
- `handler/` → can import `usecase`, `service`, `domain`
- `middleware/` → can import `domain`
- `router/` → can import `handler`, `middleware`
- `internal/params/` → can import `domain` (for DTOs)

---

## Common Commands

### Running the Application
```bash
# Run the API server
make run

# Run development services (PostgreSQL, Swagger UI)
make run-dev

# Stop development services
make stop-dev

# Refresh Swagger UI after updating docs
make swagger
```

### Database Migrations
```bash
# Migrations are in migrations/ directory
# Use golang-migrate or goose to run them
# Migration files follow pattern: NNN_name.up.sql and NNN_name.down.sql
```

### Wire (Dependency Injection)
```bash
# Regenerate Wire dependencies after modifying wire.go
go generate ./internal/infrastructure/...
```

---

## API Reference

### Authentication Endpoints
- `POST /api/v1/auth/register` - Register with OTP
- `POST /api/v1/auth/verify-otp` - Verify OTP
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/logout` - Logout

### Contacts Endpoints
- `GET /api/v1/contacts` - Get all contacts
- `POST /api/v1/contacts` - Create contact
- `PUT /api/v1/contacts/{id}` - Update contact
- `DELETE /api/v1/contacts/{id}` - Delete contact
- `POST /api/v1/contacts/bulk` - Bulk operations

### Sessions Endpoints
- `GET /api/v1/sessions` - Get all sessions
- `POST /api/v1/sessions` - Create session
- `GET /api/v1/sessions/{id}` - Get session details
- `PUT /api/v1/sessions/{id}` - Update session
- `DELETE /api/v1/sessions/{id}` - Cancel session
- `POST /api/v1/sessions/{id}/participants` - Add participants
- `POST /api/v1/sessions/{id}/bills` - Add bill item
- `PUT /api/v1/sessions/{id}/calculate-splits` - Calculate splits

### Payments Endpoints
- `POST /api/v1/payments/{participant_id}/submit` - Submit proof
- `POST /api/v1/payments/{proof_id}/approve` - Approve payment
- `POST /api/v1/payments/{proof_id}/reject` - Reject payment

### Notifications Endpoints
- `POST /api/v1/sessions/{id}/send-notifications` - Send notifications
- `POST /api/v1/participants/{participant_id}/reminder` - Send reminder

---

## Implementation Guide

For detailed implementation phases and step-by-step workflow, see:

**`~/.openclaw/workspace/letpai-backend-dev-guide.md`**

This guide contains:
- Complete 6-phase implementation plan
- Current implementation status
- Tasks for each phase
- Dependencies between phases

### Quick Reference

The dev guide covers:
1. **Phase 1**: Domain Layer (entities, ports, value objects)
2. **Phase 2**: Data Access Layer (repository implementations)
3. **Phase 3**: Business Logic Layer (use cases, services)
4. **Phase 4**: HTTP Interface Layer (handlers, middleware, router)
5. **Phase 5**: Dependency Injection (Wire setup)
6. **Phase 6**: Entry Point (main.go, server startup)

---

## Testing Guidelines

### Unit Tests
- Test `domain/` layer (no external dependencies)
- Test `usecase/` layer (mock repositories)
- Test `service/` layer (pure business logic)

### Integration Tests
- Test `repository/` layer (real database)
- Test `handler/` layer (HTTP requests)

---

## Common Patterns

### Error Handling

Use domain errors in `domain/`:
```go
var (
    ErrBadRequest = errors.New("BAD_REQUEST")
    ErrNotFound = errors.New("NOT_FOUND")
    ErrUnauthorized = errors.New("UNAUTHORIZED")
    ErrForbidden = errors.New("FORBIDDEN")
    ErrConflict = errors.New("CONFLICT")
    ErrInternalFailure = errors.New("INTERNAL_FAILURE")
)

func NewError(serviceErr error, appErr string) error {
    return fmt.Errorf("%s: %v", appErr, serviceErr)
}
```

### HTTP Error Conversion

Use `httperror.FromError()` in handlers to convert domain errors to HTTP responses:
```go
func FromError(err error) (int, interface{}) {
    switch err {
    case domain.ErrNotFound:
        return fiber.StatusNotFound, map[string]interface{}{"error": "Not found"}
    case domain.ErrBadRequest:
        return fiber.StatusBadRequest, map[string]interface{}{"error": "Bad request"}
    default:
        return fiber.StatusInternalServerError, map[string]interface{}{"error": "Internal server error"}
    }
}
```

### Fiber JSON Response

Use Fiber's JSON helper for responses:
```go
return c.JSON(fiber.StatusOK, fiber.Map{
    "success": true,
    "data":    data,
})
```

---

## Important Notes

- **NEVER** modify database schema directly - use migrations
- **ALWAYS** read brief files before implementing features
- **USE** Context7 for library documentation
- **FOLLOW** Hexagonal Architecture dependency rules
- **REFERENCE** swagger.yaml for API contracts
- **domain/** is at root level, NOT in internal/
- **domain/ports/** contains repository interfaces (not implementations)
