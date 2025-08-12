FROM golang:1.24.6-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY . .
WORKDIR /app/tools/memoryctl
RUN GOWORK=off CGO_ENABLED=0 GOOS=linux go build -o /app/memoryctl .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/memoryctl ./memoryctl
RUN apk add --no-cache curl
ENTRYPOINT ["./memoryctl"]


