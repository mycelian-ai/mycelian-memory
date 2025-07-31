# syntax=docker/dockerfile:1
# --- Build stage ---
FROM golang:1.24.5-alpine AS builder

WORKDIR /app

# Install git (needed for go modules that fetch private repos)
RUN apk add --no-cache git

# Copy go mod files and download deps first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the memoryctl binary (static)
RUN CGO_ENABLED=0 GOOS=linux go build -o memoryctl ./cmd/memoryctl

# --- Runtime stage ---
FROM alpine:latest

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/memoryctl ./memoryctl

# Small base image additional utils
RUN apk add --no-cache curl

ENTRYPOINT ["./memoryctl"] 