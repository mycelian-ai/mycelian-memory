FROM golang:1.24.6-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY . .
WORKDIR /app/tools/mycelian-service-tools
RUN GOWORK=off CGO_ENABLED=0 GOOS=linux go build -o /app/mycelian-service-tools .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/mycelian-service-tools ./mycelian-service-tools
RUN apk add --no-cache curl
ENTRYPOINT ["./mycelian-service-tools"]
