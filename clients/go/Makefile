# Variables
BINARY_NAME=synapse-mcp-server
CMD_PATH=./cmd/synapse-mcp-server
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
CONTAINER_NAME=synapse-mcp-dev

# Build targets
.PHONY: build
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) $(CMD_PATH)

.PHONY: build-all
build-all:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 $(CMD_PATH)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 $(CMD_PATH)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)

.PHONY: install
install:
	go install $(LDFLAGS) $(CMD_PATH)

# Test targets
.PHONY: test
test:
	go test ./...

.PHONY: test-cover
test-cover:
	go test -cover ./...

.PHONY: test-race
test-race:
	go test -race ./...

.PHONY: test-integration
test-integration:
	go test -tags=integration ./...

.PHONY: test-cli-docker
test-cli-docker: docker-build
	docker run --rm $(BINARY_NAME):latest ./synapse --help | grep -q "Synapse CLI" && echo "✅ CLI smoke test passed"

# Development targets
.PHONY: run
run:
	go run $(CMD_PATH)

.PHONY: dev
dev:
	go run -race $(CMD_PATH)

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: tidy
tidy:
	go mod tidy
	go mod verify

# Docker targets
.PHONY: docker-build
docker-build:
	docker build -t $(BINARY_NAME):latest .
	docker tag $(BINARY_NAME):latest $(BINARY_NAME):$(VERSION)

.PHONY: docker-run
docker-run:
	docker run --rm -i $(BINARY_NAME):latest

# Development Docker targets
.PHONY: docker-dev-build
docker-dev-build:
	docker build -t $(BINARY_NAME):dev .

.PHONY: docker-dev-kill
docker-dev-kill:
	@docker kill $(CONTAINER_NAME) 2>/dev/null || true
	@docker rm $(CONTAINER_NAME) 2>/dev/null || true

.PHONY: docker-dev-run
docker-dev-run: docker-dev-kill docker-dev-build
	docker run --name $(CONTAINER_NAME) --rm -i \
		-e MEMORY_SERVICE_URL=http://host.docker.internal:8080 \
		$(BINARY_NAME):dev

.PHONY: docker-dev-restart
docker-dev-restart: docker-dev-kill docker-dev-build docker-dev-run

.PHONY: docker-dev-logs
docker-dev-logs:
	docker logs -f $(CONTAINER_NAME)

# Quick development iteration
.PHONY: dev-iterate
dev-iterate: build docker-dev-restart

# Docker Compose targets
.PHONY: compose-up
compose-up:
	$(MAKE) -C ../../ mcp-streamable-up

.PHONY: compose-down
compose-down:
	$(MAKE) -C ../../ mcp-streamable-down

.PHONY: compose-restart
compose-restart: compose-down compose-up

.PHONY: compose-logs
compose-logs:
	docker compose -f ../../deployments/docker/docker-compose.client.mcp.yml logs -f

# Fast Docker development iteration
.PHONY: dev-reload
dev-reload:
	@echo "🔄 Rebuilding Docker image..."
	docker build -t $(BINARY_NAME):dev .
	@echo "✅ Ready for Claude Desktop"

.PHONY: dev-status
dev-status:
	@echo "Docker Image: $(shell docker images -q $(BINARY_NAME):dev >/dev/null 2>&1 && echo '✅ Built' || echo '❌ Missing')"
	@echo "Memory Service: $(shell curl -s http://localhost:8080/health >/dev/null 2>&1 && echo '✅ Running' || echo '❌ Down')"

# Clean targets
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	go clean

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          Build the binary for current platform"
	@echo "  build-all      Build binaries for all platforms"
	@echo "  install        Install the binary to GOPATH/bin"
	@echo "  test           Run tests"
	@echo "  test-cover     Run tests with coverage"
	@echo "  test-race      Run tests with race detection"
	@echo "  test-integration Run integration tests"
	@echo "  run            Run the application"
	@echo "  dev            Run the application with race detection"
	@echo "  fmt            Format Go code"
	@echo "  vet            Run go vet"
	@echo "  lint           Run golangci-lint"
	@echo "  tidy           Tidy and verify go modules"
	@echo "  docker-build   Build Docker image"
	@echo "  docker-run     Run Docker container"
	@echo "  docker-dev-build Build Docker development image"
	@echo "  docker-dev-kill Kill Docker development container"
	@echo "  docker-dev-run Run Docker development container"
	@echo "  docker-dev-restart Restart Docker development container"
	@echo "  docker-dev-logs View Docker development logs"
	@echo "  dev-iterate    Build and run Docker development container"
	@echo "  compose-up     Start Docker Compose services"
	@echo "  compose-down   Stop Docker Compose services"
	@echo "  compose-restart Restart Docker Compose services"
	@echo "  compose-logs   View Docker Compose logs"
	@echo "  dev-reload     Rebuild Docker development image"
	@echo "  dev-status     Check Docker development status"
	@echo "  clean          Clean build artifacts"
	@echo "  help           Show this help message" 