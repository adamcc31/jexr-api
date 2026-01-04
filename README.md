# Go Recruitment Backend

A clean architecture backend for recruitment system using Golang, Gin, Postgres (pgx), and Supabase Auth (JWKS).

## Prerequisites

- Go 1.21+
- PostgreSQL
- Supabase Project (URL and Anon/Service Key if needed, but mainly JWKS URL)

## Setup

1. **Install Dependencies**
   ```bash
   go mod tidy
   ```
   *Note: If `go` command was not found during generation, please ensure Go is installed and in your PATH.*

2. **Environment Variables**
   Create `.env` file:
   ```env
   PORT=8080
   DATABASE_URL=postgres://user:pass@localhost:5432/dbname?sslmode=disable
   SUPABASE_URL=https://your-project.supabase.co
   ```

3. **Database Migration**
   Run the SQL script in `migrations/000001_init.up.sql` against your Postgres database.

4. **Swagger Documentation**
   Install swaggo:
   ```bash
   go install github.com/swaggo/swag/cmd/swag@latest
   ```
   Generate docs:
   ```bash
   swag init -g cmd/api/main.go
   ```

5. **Run**
   ```bash
   go run cmd/api/main.go
   ```

## Architecture

- **cmd/api**: Entrypoint
- **config**: Configuration loading
- **internal**: Private application code
    - **domain**: Business entities and interfaces (Pure Go)
    - **usecase**: Business logic
    - **repository**: Data access (Postgres/pgx)
    - **delivery**: HTTP handlers and middleware (Gin)
- **pkg**: Public shared code (Logger, Response, Auth/JWKS)

## API Endpoints

- `GET /v1/health`: Health check
- `POST /v1/auth/sync`: Sync Supabase user to local DB
- `GET /v1/auth/me`: Get current user profile
- `POST /v1/jobs`: Create job (Auth required)
- `GET /v1/jobs`: List jobs
- `GET /v1/jobs/:id`: Job details

## Security Features

### 1. Redis-Backed Rate Limiting
- **Global Limit**: 100 requests/minute per IP (configurable via `RATE_LIMIT_GLOBAL_THRESHOLD`).
- **Auth Endpoint Limit**: 10 requests/minute per IP (configurable).
- **Login Endpoint Limit**: 5 attempts/minute per IP.
- **Provider**: Upstash Redis (configured via `UPSTASH_REDIS_URL`).
- **Fallback**: In-memory rate limiting if Redis is unavailable (fail-open for general, fail-closed for auth).

### 2. Failed Login Blocking
- **Threshold**: 5 failed attempts in 15 minutes triggers a temp block.
- **Block Duration**: 15 minutes (configurable via `FAILED_LOGIN_BLOCK_MINUTES`).
- **Tracking**: Redis keys `fail:login:user:<email>` and `blocked:login:user:<email>`.

### 3. Security Logging (Audit Trail)
- **Format**: Structured JSON via Zap logger.
- **Events Logged**: `login_failed`, `login_blocked`, `rate_limit_triggered` (with error details).
- **Persistence**: Logs are persisted to `security_events` table (90-day retention policy recommended).
- **PII Protection**: Sensitive fields (emails) are masked or hashed.

### 4. Input Validation
- **Candidate Profile**: Strict validation for names (no numbers/emoji) and bio.
- **Responses**: Structured 400 errors with specific field validation messages.

### Configuration (Environment Variables)
```bash
# Redis
UPSTASH_REDIS_URL=rediss://default:password@...
UPSTASH_REDIS_PASSWORD=...

# Rate Limiting
RATE_LIMIT_WINDOW_SECONDS=60
RATE_LIMIT_GLOBAL_THRESHOLD=100
RATE_LIMIT_LOGIN_THRESHOLD=5
FAILED_LOGIN_MAX_ATTEMPTS=5
FAILED_LOGIN_BLOCK_MINUTES=15

# Security Logging
SECURITY_LOG_TO_DB=true
```
