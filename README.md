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
