# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Install git dan sertifikat SSL
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy seluruh source code
COPY . .

# --- BAGIAN KRITIS YANG DIPERBAIKI ---
# Sebelumnya: go build -o recruitment-backend main.go (SALAH)
# Sekarang: Arahkan ke folder cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o recruitment-backend ./cmd/api

# Stage 2: Run
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy binary dari builder
COPY --from=builder /app/recruitment-backend .

# Copy folder .env jika diperlukan (opsional, sebaiknya pakai ENV vars di Railway)
# COPY --from=builder /app/.env .

# Jalankan binary
CMD ["./recruitment-backend"]
