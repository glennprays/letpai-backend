# AGENTS.md - Letpai Backend Development

**This file provides guidance to AI agents when working with code in this repository.**

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
- **DI Framework**: Google Wire for dependency injection

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
   - Core flows (auth, contacts, sessions, payments, admin)
   - UI/UX requirements

3. **Database Schema**: `~/.openclaw/workspace/letpai-database-schema.md`
   - Complete database schema with all tables (including admin tables)
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

### Active Task Tracking

When working on multi-phase features, check for TODO files:
- **`ADMIN_TODO.md`** - Remaining tasks for admin module implementation
- Currently in active development

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
├── entity/              # Domain entities (Admin, User, Session, etc.)
├── ports/               # Repository interfaces
└── valueobject/         # Value objects
internal/
├── config/              # Configuration loading
├── usecase/             # Application business rules
│   ├── auth/            # Authentication use cases
│   ├── admin/           # Admin use cases (partially implemented)
│   ├── contact/         # Contact management
│   ├── session/         # Bill splitting sessions
│   ├── payment/         # Payment processing
│   ├── notification/    # WhatsApp notifications
│   └── dashboard/      # Dashboard statistics
├── service/             # Reusable services (JWT, WhatsApp, etc.)
├── repository/          # Data access implementations
│   ├── admin_postgres.go         # Admin repository
│   ├── user_postgres.go         # User repository
│   └── whatsapp_config_postgres.go  # WhatsApp config repository
├── handler/             # HTTP handlers
│   ├── admin_handler.go           # Admin endpoints
│   ├── auth_handler.go           # Auth endpoints
│   └── ...
├── middleware/          # HTTP middleware (auth, rate limiting, etc.)
├── router/              # Route definitions
├── httperror/           # HTTP error conversion
├── infrastructure/      # Wire DI, DB, Redis
├── params/              # Request/Response DTOs
│   ├── request/admin/    # Admin request types
│   ├── response/admin/   # Admin response types
│   ├── request/
│   └── response/
└── utils/               # Helper utilities
pkg/
└── logger/              # Shared logger
migrations/             # Database migration files
├── 000010_create_admins.up.sql
├── 000011_create_whatsapp_configs.up.sql
├── 000012_create_admin_ops_verifications.up.sql
└── 000013_seed_super_admin.up.sql
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
go run github.com/google/wire/cmd/wire ./internal/infrastructure
```

### Build
```bash
# Build the application
go build -o bin/letpai-api ./cmd/api/main.go
```

---

## API Reference

### Authentication Endpoints
- `POST /api/v1/auth/register` - Register with OTP
- `POST /api/v1/auth/verify-otp` - Verify OTP
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/logout` - Logout
- `POST /api/v1/auth/profile` - Update profile

### Admin Endpoints (Partially Implemented)
**Public Admin Auth Routes:**
- `POST /api/v1/admin/auth/initiate` - Initiate admin login flow
- `POST /api/v1/admin/auth/login` - Admin login with OTP or password
- `POST /api/v1/admin/auth/verify-otp` - Verify OTP code

**Protected Admin Routes:**
- `GET /api/v1/admin/profile` - Get current admin profile
- `PUT /api/v1/admin/profile/setup-password` - Set initial password
- `GET /api/v1/admin/status` - Get WhatsApp gateway status
- `POST /api/v1/admin/qr-code` - Generate QR code for WhatsApp pairing
- `POST /api/v1/admin/logout` - Admin logout
- `PUT /api/v1/admin/config` - Update WhatsApp config

**Admin Management (Super Admin Only):**
- `GET /api/v1/admin/admins` - List all admins
- `POST /api/v1/admin/admins` - Create new admin
- `PUT /api/v1/admin/admins/:id` - Update admin
- `DELETE /api/v1/admin/admins/:id` - Delete admin

### Contacts Endpoints
- `GET /api/v1/contacts` - Get all contacts
- `POST /api/v1/contacts` - Create contact
- `PUT /api/v1/contacts/{id}` - Update contact
- `DELETE /api/v1/contacts/{id}` - Delete contact
- `POST /api/v1/contacts/bulk` - Bulk operations
- `POST /api/v1/contacts/import` - Import from contact groups

### Contact Groups Endpoints
- `GET /api/v1/contact-groups` - Get all groups
- `POST /api/v1/contact-groups` - Create group
- `PUT /api/v1/contact-groups/{id}` - Update group
- `DELETE /api/v1/contact-groups/{id}` - Delete group

### Sessions Endpoints
- `GET /api/v1/sessions` - Get all sessions
- `POST /api/v1/sessions` - Create session
- `GET /api/v1/sessions/{id}` - Get session details
- `PUT /api/v1/sessions/{id}` - Update session
- `DELETE /api/v1/sessions/{id}` - Cancel session
- `POST /api/v1/sessions/{id}/participants` - Add participants
- `DELETE /api/v1/sessions/{id}/participants/:participant_id` - Remove participant
- `PUT /api/v1/sessions/{id}/participants/:participant_id` - Update participant
- `POST /api/v1/sessions/{id}/bills` - Add bill item
- `PUT /api/v1/sessions/{id}/bills/:bill_item_id` - Update bill item
- `DELETE /api/v1/sessions/{id}/bills/:bill_item_id` - Delete bill item
- `PUT /api/v1/sessions/{id}/calculate-splits` - Calculate splits

### Payments Endpoints
**Public Routes:**
- `POST /api/v1/payments/{participant_id}/submit` - Submit payment proof
- `GET /api/v1/payments/{participant_id}/public` - Get public payment page

**Protected Routes:**
- `POST /api/v1/payments/{proof_id}/approve` - Approve payment
- `POST /api/v1/payments/{proof_id}/reject` - Reject payment
- `POST /api/v1/payments/bulk-approve` - Bulk approve payments
- `POST /api/v1/payments/bulk-reject` - Bulk reject payments

### Notifications Endpoints
- `POST /api/v1/sessions/{id}/send-notifications` - Send notifications to participants
- `POST /api/v1/sessions/{id}/bulk-reminder` - Bulk reminders
- `POST /api/v1/participants/{participant_id}/reminder` - Send individual reminder

### Dashboard Endpoints
- `GET /api/v1/dashboard` - Get dashboard statistics

### Webhook Endpoints
- `POST /api/v1/webhooks/whatsapp-status` - WhatsApp status webhook

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

### Admin Module Status

**Completed:**
- ✅ Database migrations (admins, whatsapp_configs, admin_ops_verifications tables)
- ✅ Domain entities (Admin, WhatsAppConfig)
- ✅ Repository interfaces and PostgreSQL implementations
- ✅ Admin handler with placeholder implementations
- ✅ Admin routes in router (13 endpoints)
- ✅ Request/response DTOs
- ✅ Wire DI setup updated

**Remaining (see ADMIN_TODO.md):**
- ⏳ Complete admin use cases
- ⏳ Admin authorization middleware (role-based)
- ⏳ Replace handler placeholders with real implementations
- ⏳ Complete repository methods
- ⏳ Admin operations verification flow
- ⏳ WhatsApp service integration
- ⏳ Testing
- ⏳ Documentation
- ⏳ Security hardening

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

### Use Case Pattern

Follow the pattern in `internal/usecase/auth/`:
```go
type SomeUseCase struct {
    userRepo ports.UserRepository
    // Add dependencies
}

func NewSomeUseCase(userRepo ports.UserRepository) *SomeUseCase {
    return &SomeUseCase{userRepo: userRepo}
}

func (uc *SomeUseCase) Execute(ctx context.Context, req *SomeRequest) (*SomeResponse, error) {
    // Business logic here
}
```

### Repository Pattern

Implement repository interface from `domain/ports/`:
```go
type PostgresSomeRepository struct {
    db *sqlx.DB
}

func NewPostgresSomeRepository(db *sqlx.DB) ports.SomeRepository {
    return &PostgresSomeRepository{db: db}
}

// Implement all interface methods
func (r *PostgresSomeRepository) Create(ctx context.Context, entity *entity.Some) error {
    // Implementation
}
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
- **Wire** is used for dependency injection - regenerate after changes
- **Admin module** is partially implemented - check ADMIN_TODO.md for status
- **Role-based access** - Admin vs Super Admin roles need enforcement

---

## File Structure Notes

### Admin Module Files
- `domain/entity/admin.go` - Admin entity
- `domain/ports/admin_repository.go` - Admin repository interface
- `internal/repository/admin_postgres.go` - PostgreSQL implementation
- `internal/handler/admin_handler.go` - HTTP handlers
- `internal/params/request/admin/` - Request DTOs
- `internal/params/response/admin/` - Response DTOs
- `migrations/000010_create_admins.up.sql` - Admin table migration
- `migrations/000011_create_whatsapp_configs.up.sql` - WhatsApp config migration
- `migrations/000012_create_admin_ops_verifications.up.sql` - Verification log migration
- `migrations/000013_seed_super_admin.up.sql` - Super admin seed data

### WhatsApp Integration Files
- `domain/entity/whatsapp_config.go` - WhatsApp config entity
- `domain/ports/whatsapp_api_config_repository.go` - Repository interface
- `internal/repository/whatsapp_config_postgres.go` - PostgreSQL implementation
- `internal/service/whatsapp.go` - WhatsApp service wrapper
