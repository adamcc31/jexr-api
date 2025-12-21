# Stage 1: Build
FROM golang:1.23-alpine AS builder

# Install git dan sertifikat SSL
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Enable automatic Go toolchain management
ENV GOTOOLCHAIN=auto

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy seluruh source code
COPY . .

# Build binary - CGO_ENABLED=0 membuat binary statis (tidak butuh dependency OS)
RUN CGO_ENABLED=0 GOOS=linux go build -o recruitment-backend ./cmd/api

# Stage 2: Run
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy binary dari builder
COPY --from=builder /app/recruitment-backend .

# Expose port for Koyeb
EXPOSE 8080

# Jalankan binary
CMD ["./recruitment-backend"]
