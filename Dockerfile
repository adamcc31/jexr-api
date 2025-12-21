# --- Stage 1: Build ---
FROM golang:1.22-alpine AS builder

# Install git dan certificates (penting untuk HTTPS request ke Supabase/DB)
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy dependency files dulu agar ter-cache layer-nya
COPY go.mod go.sum ./
RUN go mod download

# Copy source code sisanya
COPY . .

# Build binary. 
# CGO_ENABLED=0 membuat binary statis (tidak butuh dependency OS)
RUN CGO_ENABLED=0 GOOS=linux go build -o recruitment-backend main.go

# --- Stage 2: Production Run ---
FROM alpine:latest

# Install CA Certs agar bisa connect ke HTTPS (Supabase/Postgres SSL)
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary dari Stage 1
COPY --from=builder /app/recruitment-backend .
# Copy file .env jika aplikasi anda WAJIB butuh file fisik (tapi sebaiknya pakai ENV VAR di Railway)
# COPY --from=builder /app/.env . 

# Expose port (hanya dokumentasi, Railway akan override)
EXPOSE 8080

# Command untuk menjalankan aplikasi
CMD ["./recruitment-backend"]