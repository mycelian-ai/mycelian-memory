# Multi-stage build for memory-service
FROM golang:1.24.6-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

# Copy entire repository (for replace ../../server)
COPY . .

# Build within the module directory to avoid go.work workspace coupling
WORKDIR /app/cmd/memory-service
RUN GOWORK=off CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/memory-service .

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/
COPY --from=builder /app/memory-service .

EXPOSE 11545 9090
CMD ["./memory-service"]
