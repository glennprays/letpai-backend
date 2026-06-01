# Letpai Backend

> Split bills with friends via WhatsApp

Backend API for Letpai — a bill splitting app with WhatsApp integration. Handles auth, session management, split calculations, payment tracking, and WhatsApp notification dispatch.

## Features

- **Auth** — WhatsApp OTP + password login, JWT tokens
- **Sessions** — CRUD with participants, public slugs for shareable links
- **Bill Splitting** — Per-item service charge & tax, fee-aware split calculation
- **Payments** — Multi-bank accounts, proof upload (S3), mark-as-paid
- **WhatsApp Notifications** — Template messages via WAGA gateway, reminders, retry
- **Admin** — Bootstrap CLI, password auth, team management, gateway control
- **Observability** — Structured logging, panic recovery with stack traces

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.25 |
| HTTP | Fiber |
| Database | PostgreSQL |
| Cache | Redis (OTP, rate limiting) |
| Storage | S3 (proof images) |
| DI | Wire |
| Migrations | golang-migrate |
| Deployment | Docker |

## Getting Started

### Prerequisites

- Go 1.25+
- PostgreSQL 15+
- Redis 7+
- Docker (optional)

### Setup

```bash
# Clone
git clone https://github.com/glennprays/letpai-backend.git
cd letpai-backend

# Configure
cp .env.example .env
# Edit .env with your database, Redis, S3, and WhatsApp gateway credentials

# Run migrations
make migrate-up

# Run
go run cmd/api/main.go
```

### Docker

```bash
docker build -t letpai-backend .
docker run -p 3000:3000 --env-file .env letpai-backend
```

## Project Structure

```
internal/
├── config/          # Environment config (Viper)
├── domain/          # Core entities & interfaces
├── usecase/         # Application business rules
├── service/         # Business logic services
├── repository/      # Data access (PostgreSQL, Redis)
├── handler/         # HTTP handlers (Fiber)
├── middleware/       # Auth, CORS, logging
├── router/          # Route definitions
├── httperror/       # Error response mapping
├── infrastructure/  # DI container (Wire)
├── utils/           # Helpers
└── worker/          # Background jobs
migrations/          # Database migrations
```

## License

Private — © 2025 Glenn Pray
