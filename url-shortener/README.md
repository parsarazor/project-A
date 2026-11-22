# URL Shortener Service

A professional URL shortener service built with Go, following industry best practices for backend development.

## 📋 Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Tech Stack](#tech-stack)
- [Architecture](#architecture)
- [Database Schema](#database-schema)
- [API Documentation](#api-documentation)
- [Getting Started](#getting-started)
- [Configuration](#configuration)
- [Testing](#testing)
- [Docker](#docker)
- [Project Structure](#project-structure)

## Overview

This URL Shortener service converts long URLs into short, manageable links. It provides user authentication, link management, and administrative capabilities. The service is designed with security, reliability, and scalability in mind.

## Features

### Core Features
- ✅ Convert long URLs to short codes
- ✅ HTTP 302 redirects
- ✅ URL validation
- ✅ Unique short code generation
- ✅ Click/visit tracking

### Authentication & Authorization
- ✅ User registration and login
- ✅ JWT-based authentication
- ✅ Password hashing (bcrypt)
- ✅ Access token management
- ✅ Role-based access control (User/Admin)

### API Features
- ✅ RESTful API design
- ✅ API versioning (`/api/v1`)
- ✅ Standard HTTP status codes
- ✅ Consistent error handling
- ✅ Request validation

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.21+ |
| Framework | Fiber v2 |
| Database | PostgreSQL 15 |
| ORM | GORM |
| Authentication | JWT |
| Password Hashing | bcrypt |
| Configuration | Viper |
| Containerization | Docker & Docker Compose |

### Why These Choices?

- **Fiber**: High-performance web framework with Express-like syntax, excellent middleware support
- **PostgreSQL**: Robust, ACID-compliant relational database perfect for structured data
- **GORM**: Feature-rich ORM with migrations, relationships, and hooks support
- **JWT**: Industry-standard, stateless authentication mechanism

## Architecture

The project follows a **hexagonal architecture (ports and adapters)** in a **single monolith**.

```
┌──────────────────────────────────────────────────────────────┐
│                  Primary Adapter (HTTP/Fiber)               │
│         handlers + middleware map requests to use cases      │
└──────────────────────────────┬───────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                Core (Application + Domain Rules)            │
│   internal/core/usecase + internal/core/ports contracts     │
└──────────────────────────────┬───────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────┐
│      Secondary Adapters (DB/JWT/Bcrypt/ShortCode)           │
│ repository (GORM), adapter/security, adapter/shortcode      │
└──────────────────────────────────────────────────────────────┘
```

### What Matters

- **Dependency direction**: adapters depend on core, core does not depend on Fiber/GORM.
- **Ports**: interfaces in `internal/core/ports` define what core needs.
- **Use cases**: business rules live in `internal/core/usecase`.
- **Composition root**: wiring is done in `cmd/*/main.go`.

## Database Schema

### Entity Relationship Diagram

```
┌──────────────────┐       ┌──────────────────┐
│      users       │       │       urls       │
├──────────────────┤       ├──────────────────┤
│ id (PK)          │───┐   │ id (PK)          │
│ email            │   │   │ short_code       │
│ password_hash    │   │   │ original_url     │
│ name             │   └──>│ user_id (FK)     │
│ role             │       │ click_count      │
│ created_at       │       │ is_active        │
│ updated_at       │       │ created_at       │
└──────────────────┘       │ updated_at       │
                           └──────────────────┘
```

### Tables

#### Users Table
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique identifier |
| email | VARCHAR(255) | UNIQUE, NOT NULL | User email |
| password_hash | VARCHAR(255) | NOT NULL | Bcrypt hashed password |
| name | VARCHAR(100) | NOT NULL | User display name |
| role | VARCHAR(20) | NOT NULL, DEFAULT 'user' | User role (user/admin) |
| created_at | TIMESTAMP | NOT NULL | Creation timestamp |
| updated_at | TIMESTAMP | NOT NULL | Last update timestamp |

#### URLs Table
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique identifier |
| short_code | VARCHAR(10) | UNIQUE, NOT NULL | Generated short code |
| original_url | TEXT | NOT NULL | Original long URL |
| user_id | UUID | FOREIGN KEY | Owner of the URL |
| click_count | BIGINT | DEFAULT 0 | Number of redirects |
| is_active | BOOLEAN | DEFAULT true | URL active status |
| created_at | TIMESTAMP | NOT NULL | Creation timestamp |
| updated_at | TIMESTAMP | NOT NULL | Last update timestamp |

### Indexes
- `idx_urls_short_code` on `urls.short_code`
- `idx_urls_user_id` on `urls.user_id`
- `idx_users_email` on `users.email`

## API Documentation

### Base URL
```
http://localhost:8080/api/v1
```

### Authentication
All protected endpoints require a Bearer token in the Authorization header:
```
Authorization: Bearer <access_token>
```

### Endpoints

#### Authentication

##### Register User
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securePassword123",
  "name": "John Doe"
}
```

**Response (201 Created)**
```json
{
  "success": true,
  "message": "User registered successfully",
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "John Doe",
    "role": "user",
    "created_at": "2024-01-15T10:00:00Z"
  }
}
```

##### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securePassword123"
}
```

**Response (200 OK)**
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 86400
  }
}
```

#### URL Management

##### Create Short URL
```http
POST /api/v1/urls
Authorization: Bearer <token>
Content-Type: application/json

{
  "original_url": "https://example.com/very/long/url/path"
}
```

**Response (201 Created)**
```json
{
  "success": true,
  "message": "URL shortened successfully",
  "data": {
    "id": "uuid",
    "short_code": "abc123",
    "short_url": "http://localhost:8080/abc123",
    "original_url": "https://example.com/very/long/url/path",
    "click_count": 0,
    "created_at": "2024-01-15T10:00:00Z"
  }
}
```

##### Get User's URLs
```http
GET /api/v1/urls/me
Authorization: Bearer <token>
```

**Response (200 OK)**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "short_code": "abc123",
      "short_url": "http://localhost:8080/abc123",
      "original_url": "https://example.com/...",
      "click_count": 42,
      "is_active": true,
      "created_at": "2024-01-15T10:00:00Z"
    }
  ]
}
```

##### Redirect (Public)
```http
GET /{shortCode}
```

**Response (302 Found)**
Redirects to the original URL.

#### Admin Endpoints

##### Get All URLs (Admin)
```http
GET /api/v1/admin/urls
Authorization: Bearer <admin_token>
```

**Response (200 OK)**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "short_code": "abc123",
      "original_url": "https://example.com/...",
      "user": {
        "id": "uuid",
        "email": "user@example.com",
        "name": "John Doe"
      },
      "click_count": 42,
      "is_active": true,
      "created_at": "2024-01-15T10:00:00Z"
    }
  ]
}
```

##### Delete URL (Admin)
```http
DELETE /api/v1/admin/urls/{id}
Authorization: Bearer <admin_token>
```

**Response (200 OK)**
```json
{
  "success": true,
  "message": "URL deleted successfully"
}
```

### Error Responses

All errors follow this format:
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message"
  }
}
```

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | INVALID_REQUEST | Invalid request body or parameters |
| 401 | UNAUTHORIZED | Missing or invalid authentication |
| 403 | FORBIDDEN | Insufficient permissions |
| 404 | NOT_FOUND | Resource not found |
| 409 | CONFLICT | Resource already exists |
| 422 | VALIDATION_ERROR | Input validation failed |
| 500 | INTERNAL_ERROR | Internal server error |

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- Make (optional, for using Makefile commands)

### Quick Start with Docker

```bash
# Clone the repository
git clone <repository-url>
cd url-shortener

# Copy environment file
cp .env.example .env

# Start all services
docker-compose up -d

# The API will be available at http://localhost:8080
```

### Local Development

```bash
# Install dependencies
go mod download

# Set up environment variables
cp .env.example .env
# Edit .env with your configuration

# Run database migrations
go run cmd/migrate/main.go

# Start the server with hot reload (requires air)
air

# Or start without hot reload
go run cmd/api/main.go
```

## Configuration

Configuration is managed through environment variables or a config file.

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_ENV` | Environment (development/staging/production) | development |
| `APP_PORT` | Server port | 8080 |
| `APP_BASE_URL` | Base URL for short links | http://localhost:8080 |
| `DB_HOST` | Database host | localhost |
| `DB_PORT` | Database port | 5432 |
| `DB_USER` | Database user | postgres |
| `DB_PASSWORD` | Database password | postgres |
| `DB_NAME` | Database name | urlshortener |
| `DB_SSLMODE` | Database SSL mode | disable |
| `JWT_SECRET` | Secret key for JWT signing | (required) |
| `JWT_EXPIRY` | Token expiry duration | 24h |
| `TRUSTED_PROXIES` | Comma-separated proxy IPs/CIDRs trusted to set `X-Forwarded-For` (empty = trust nobody) | (empty) |

### Example .env File

```env
APP_ENV=development
APP_PORT=8080
APP_BASE_URL=http://localhost:8080

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=urlshortener
DB_SSLMODE=disable

JWT_EXPIRY=24h
```

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/service/...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Test Structure

```
tests/
├── unit/           # Unit tests for individual components
│   ├── service/    # Service layer tests
│   └── utils/      # Utility function tests
├── integration/    # Integration tests for APIs
│   ├── auth_test.go
│   └── url_test.go
└── mocks/          # Mock implementations
```

### Test Categories

1. **Unit Tests**: Test individual functions and methods in isolation
2. **Integration Tests**: Test API endpoints with a real test database
3. **Auth Tests**: Verify authentication and authorization flows

## Docker

### Development

```bash
# Build and start all services
docker-compose up --build

# Start in detached mode
docker-compose up -d

# View logs
docker-compose logs -f app

# Stop all services
docker-compose down
```

### Production

```bash
# Build and start: migrations run automatically before the API starts
docker compose -f docker-compose.prod.yml up -d --build
```

On startup, Compose brings up three services in order:

1. `db` — Postgres, started first; the others wait for its `pg_isready`
   healthcheck.
2. `migrate` — a one-shot container from the same image running `/app/migrate`
   (schema `AutoMigrate`). The API does not start unless it exits successfully.
3. `app` — the API itself.

Re-running migrations after an image upgrade:

```bash
docker compose -f docker-compose.prod.yml run --rm migrate
```

Creating the first admin user (outside development the password must be
provided explicitly and be at least 12 characters — a run without one, or with
a publicly known default, fails fast instead of creating a weak admin):

```bash
docker compose -f docker-compose.prod.yml run --rm migrate \
  -seed-admin \
  -admin-email=admin@example.com \
  -admin-password="$(openssl rand -base64 24)"
```

The password can also be supplied through the `ADMIN_PASSWORD` environment
variable (e.g. in `.env`) instead of the `-admin-password` flag. In
`APP_ENV=development` an empty password still falls back to the local dev
default for convenience — that default is rejected everywhere else.

### Docker Commands via Makefile

```bash
make docker-build    # Build Docker image
make docker-up       # Start containers
make docker-down     # Stop containers
make docker-logs     # View logs
```

## Project Structure

```
url-shortener/
├── cmd/
│   ├── api/
│   │   └── main.go              # Application entry point
│   └── migrate/
│       └── main.go              # Database migration tool
├── internal/
│   ├── adapter/
│   │   ├── security/
│   │   │   ├── jwt_manager.go      # JWT adapter (TokenManager)
│   │   │   └── bcrypt_hasher.go    # Password hasher adapter
│   │   └── shortcode/
│   │       └── random_generator.go # Short code generator adapter
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── core/
│   │   ├── ports/
│   │   │   ├── user_repository.go  # Core contracts (ports)
│   │   │   ├── url_repository.go
│   │   │   ├── security.go
│   │   │   └── shortcode.go
│   │   └── usecase/
│   │       ├── auth_usecase.go     # Core auth business rules
│   │       └── url_usecase.go      # Core URL business rules
│   ├── database/
│   │   └── database.go          # Database connection
│   ├── handler/
│   │   ├── auth_handler.go      # Auth HTTP handlers
│   │   ├── url_handler.go       # URL HTTP handlers
│   │   └── admin_handler.go     # Admin HTTP handlers
│   ├── middleware/
│   │   ├── auth.go              # JWT authentication middleware
│   │   └── admin.go             # Admin authorization middleware
│   ├── model/
│   │   ├── user.go              # User model
│   │   └── url.go               # URL model
│   ├── repository/
│   │   ├── user_repository.go   # GORM adapter for user port
│   │   └── url_repository.go    # GORM adapter for URL port
│   ├── service/
│   │   ├── auth_service.go      # Compatibility facade over use case
│   │   └── url_service.go       # Compatibility facade over use case
│   └── utils/
│       ├── response.go          # HTTP response helpers
│       ├── validator.go         # Input validation
│       └── shortcode.go         # Short code generation
├── tests/
│   ├── unit/
│   ├── integration/
│   └── mocks/
├── .air.toml                    # Air hot reload config
├── .env.example                 # Example environment file
├── .gitignore
├── docker-compose.yml           # Development compose
├── docker-compose.prod.yml      # Production compose
├── Dockerfile                   # Development Dockerfile
├── Dockerfile.prod              # Production Dockerfile
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## License

This project is developed as part of the Quera Golang Bootcamp - Winter 1404.

---

**Happy Coding! 🚀**
