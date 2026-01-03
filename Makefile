.PHONY: build test deploy clean local-test

# Build the Go binary
build:
	@echo "Building k8flex-agent..."
	go build -o k8flex-agent ./cmd/k8flex

# Run tests
test:
	go test -v ./...

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t k8flex-agent:latest .

# Load image into Kind
kind-load: docker-build
	@echo "Loading image into Kind..."
	@CLUSTER=$$(kind get clusters 2>/dev/null | head -n1); \
	if [ -z "$$CLUSTER" ]; then \
		echo "ERROR: No Kind cluster found. Create one with: kind create cluster"; \
		exit 1; \
	fi; \
	echo "Loading into cluster: $$CLUSTER"; \
	kind load docker-image k8flex-agent:latest --name $$CLUSTER

# Load image into Minikube
minikube-load: docker-build
	@echo "Loading image into Minikube..."
	minikube image load k8flex-agent:latest

# Deploy to Kubernetes using Helmfile
helmfile-sync:
	@echo "Deploying with helmfile..."
	helmfile sync

# Preview changes before applying
helmfile-diff:
	@echo "Showing helmfile diff..."
	helmfile diff

# Deploy with helmfile (alias)
deploy: helmfile-sync
	@echo "✓ Deployed to cluster"

# Full deployment (build + load + helmfile)
deploy-kind: kind-load
	@echo "Deploying with helmfile..."
	helmfile sync
	@echo "✓ Deployed to Kind cluster"

deploy-minikube: minikube-load
	@echo "Deploying with helmfile..."
	helmfile sync
	@echo "✓ Deployed to Minikube cluster"

# Run locally (requires kubeconfig)
local-run:
	@echo "Running locally..."
	go run main.go

# Test with sample alert
local-test:
	helmfile destroy || kubectl delete namespace k8flex
	curl -XPOST 'http://localhost:8080/webhook' \
		-H 'Content-Type: application/json' \
		-d @test-alert.json

# View logs
logs:
	kubectl logs -n k8flex deployment/k8flex-agent -f

# Clean up
clean:
	rm -f k8flex-agent
	kubectl delete -f k8s/deployment.yaml --ignore-not-found=true

# Format code
fmt:
	go fmt ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Check health
health:
	@kubectl exec -n k8flex deployment/k8flex-agent -- wget -qO- http://localhost:8080/health

# Setup Slack webhook
setup-slack:
	@if [ -z "$(WEBHOOK_URL)" ]; then \
		echo "Usage: make setup-slack WEBHOOK_URL='https://hooks.slack.com/services/...'"; \
		exit 1; \
	fi
	@./setup-slack.sh '$(WEBHOOK_URL)'

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the Go binary"
	@echo "  test           - Run tests"
	@echo "  docker-build   - Build Docker image"
	@echo "  kind-load      - Build and load image into Kind"
	@echo "  minikube-load  - Build and load image into Minikube"
	@echo "  helmfile-sync  - Deploy to Kubernetes using helmfile"
	@echo "  helmfile-diff  - Preview changes before deploying"
	@echo "  deploy         - Deploy to Kubernetes (uses helmfile)"
	@echo "  deploy-kind    - Full deployment to Kind (build + load + helmfile)"
	@echo "  deploy-minikube - Full deployment to Minikube (build + load + helmfile)"
	@echo "  local-run      - Run locally"
	@echo "  local-test     - Test local instance with sample alert"
	@echo "  logs           - View agent logs"
	@echo "  clean          - Clean up resources"
	@echo "  fmt            - Format code"
	@echo "  deps           - Download dependencies"
	@echo "  health         - Check agent health"
	@echo "  setup-slack    - Setup Slack webhook: make setup-slack WEBHOOK_URL='https://...'"
