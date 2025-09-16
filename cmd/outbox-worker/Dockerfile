FROM golang:1.24.6-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY . .
WORKDIR /app/cmd/outbox-worker
RUN GOWORK=off CGO_ENABLED=0 GOOS=linux go build -o /app/outbox-worker .

FROM alpine:latest
RUN apk --no-cache add ca-certificates curl
WORKDIR /root/
COPY --from=builder /app/outbox-worker ./outbox-worker
ENTRYPOINT ["./outbox-worker"]
