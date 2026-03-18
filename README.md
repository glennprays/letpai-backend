# Golang Clean Code Architecture Starter Template

This is a starter template for building applications in Go using the Clean Architecture principles. It provides a structured way to organize your code, making it easier to maintain and scale.

## Get Started

### Prerequisites
- Go 1.25 or later
- Docker
- Make
- Git

### Clone the Repository

Clone via SSH:
```bash
git clone git@github.com:glennprays/golang-clean-arch-starter.git
```

Clone via HTTPS:
```bash
git clone https://github.com/glennprays/golang-clean-arch-starter.git
```

### Change Golang Module Name [IMPORTANT]

This step is crucial. The module name in `go.mod` should be changed to match your project name. This is important for proper dependency management and module resolution.
```bash
make rename RENAME_MODULE_TO=github.com/yourname/yourproject
```

After renaming, ensure to check the `go.mod` file to confirm the module name has been updated correctly. Then remove git history and reinitialize the repository (optional):
```bash
rm -rf .git
git init
git add .
git commit -m "Initial commit"
```

---

## Clean Architecture Overview

This template follows Clean Architecture principles, organizing code into layers with clear dependency rules.

```
┌─────────────────────────────────────────────────┐
│           Frameworks & Drivers                  │
│   handler, middleware, router, infrastructure   │
├─────────────────────────────────────────────────┤
│            Interface Adapters                   │
│           repository, httperror                 │
├─────────────────────────────────────────────────┤
│            Application Layer                    │
│             usecase, service                    │
├─────────────────────────────────────────────────┤
│              Domain Layer                       │
│                 domain                          │
└─────────────────────────────────────────────────┘
         ↑ Dependencies point inward ↑
```

### The Dependency Rule

The most important rule in Clean Architecture: **dependencies only point inward**. Inner layers know nothing about outer layers.

- `domain` → has no dependencies on other layers
- `usecase/service` → can import `domain`
- `repository` → can import `domain`
- `handler` → can import `usecase`, `service`, `domain`

---

## Directory Structure

```
├── cmd/api/                  # Application entry point
├── internal/
│   ├── config/               # Configuration loading
│   ├── domain/               # Core business entities & errors
│   ├── usecase/              # Application business rules
│   ├── service/              # Business logic services
│   ├── repository/           # Data access layer
│   ├── handler/              # HTTP handlers (Fiber)
│   ├── middleware/           # HTTP middleware
│   ├── router/               # Route definitions
│   ├── httperror/            # HTTP error responses
│   ├── infrastructure/       # DI container (Wire)
│   ├── utils/                # Helper utilities
│   └── worker/               # Background jobs
├── pkg/                      # Shared libraries
├── migrations/               # Database migrations
├── templates/                # Static templates
├── docs/                     # API documentation (Swagger)
└── misc/                     # Docker & dev configs
```

---

## Layer Explanations

### `domain/` - Domain Layer

**Purpose**: Contains core business entities, value objects, and domain-specific error types.

**Rules**:
- No external dependencies (no frameworks, no database packages)
- Define interfaces that other layers will implement
- Pure business logic only

**Example**: User entity, Order entity, domain errors

```go
// Example: domain/user.go
type User struct {
    ID    string
    Email string
    Name  string
}

// Example: domain/repository.go (interface definition)
type UserRepository interface {
    FindByID(id string) (*User, error)
    Save(user *User) error
}
```

---

### `usecase/` - Application Layer

**Purpose**: Contains application-specific business rules. Orchestrates the flow of data between entities and implements use cases.

**Rules**:
- Can import `domain`
- Cannot import `handler`, `repository` implementations
- Defines what the application does (not how)

**Example**: CreateUserUseCase, GetOrderUseCase

```go
// Example: usecase/user.go
type CreateUserUseCase struct {
    userRepo domain.UserRepository
}

func (uc *CreateUserUseCase) Execute(name, email string) (*domain.User, error) {
    // Business logic here
}
```

---

### `service/` - Business Logic Services

**Purpose**: Contains reusable business logic that can be shared across use cases.

**Rules**:
- Can import `domain`
- Used by `usecase` layer
- Encapsulates complex business operations

**Example**: EmailService, PaymentService, NotificationService

---

### `repository/` - Data Access Layer

**Purpose**: Implements data persistence. Contains database operations and external data source integrations.

**Rules**:
- Implements interfaces defined in `domain`
- Can import `domain`
- Cannot import `handler`, `usecase`

**Example**: PostgresUserRepository, RedisCache

```go
// Example: repository/user_postgres.go
type PostgresUserRepository struct {
    db *sql.DB
}

func (r *PostgresUserRepository) FindByID(id string) (*domain.User, error) {
    // Database query here
}
```

---

### `handler/` - HTTP Handlers

**Purpose**: Handles HTTP requests and responses. Converts HTTP data to domain objects and vice versa.

**Rules**:
- Can import `usecase`, `service`, `domain`
- Cannot import `repository` directly
- Handles request validation and response formatting

**Example**: UserHandler, OrderHandler

```go
// Example: handler/user.go
type UserHandler struct {
    createUser *usecase.CreateUserUseCase
}

func (h *UserHandler) Create(c *fiber.Ctx) error {
    // Parse request, call use case, return response
}
```

---

### `middleware/` - HTTP Middleware

**Purpose**: Contains middleware functions for cross-cutting concerns.

**Example**: Authentication, Logging, CORS, Rate limiting

---

### `router/` - Route Configuration

**Purpose**: Defines API routes and groups, connects handlers to endpoints.

---

### `infrastructure/` - Frameworks & DI

**Purpose**: Contains dependency injection setup (Wire), database connections, and external service configurations.

---

### `httperror/` - HTTP Error Handling

**Purpose**: Converts domain errors to appropriate HTTP responses.

---

### `utils/` - Utilities

**Purpose**: Helper functions and utilities used across the application.

---

### `worker/` - Background Jobs

**Purpose**: Background job processing, scheduled tasks, and async operations.

---

### `pkg/` - Shared Libraries

**Purpose**: Reusable packages that can be imported by other projects.

---

### `migrations/` - Database Migrations

**Purpose**: Database schema migration files.

---

## Architecture Guidelines

### 1. Dependency Injection
Use Wire for dependency injection. Define providers in `infrastructure/wire.go`.

### 2. Interface Segregation
Define small, focused interfaces in the `domain` layer. Implement them in outer layers.

### 3. Error Handling
- Define domain errors in `domain/errors.go`
- Convert to HTTP errors in `httperror/`
- Never expose internal errors to clients

### 4. Request Flow
```
HTTP Request → Handler → UseCase → Repository → Database
                  ↓          ↓           ↓
               Domain     Domain      Domain
```

### 5. Testing
- Unit test `domain` and `usecase` layers (no external dependencies)
- Integration test `repository` and `handler` layers
- Mock interfaces for isolated testing

---

## Example Implementations

### Running the Application
```bash
make run
```

### Running Development Mode
Development mode runs PostgreSQL database and Swagger API documentation using Docker containers.

To run the development mode:
```bash
make run-dev
```

To stop the development mode:
```bash
make stop-dev
```

### Updating Swagger Documentation
If you updated the swagger documentation, refresh the Swagger UI:
```bash
make swagger
```

To access the Swagger UI, open your browser and go to:
```
http://localhost:8080/
```

### API Endpoints
- `GET /health` - Health check endpoint

---

## Tech Stack

- **Framework**: [Fiber](https://gofiber.io/) - Fast HTTP framework
- **DI**: [Wire](https://github.com/google/wire) - Compile-time dependency injection
- **Database**: PostgreSQL
- **Documentation**: Swagger/OpenAPI
