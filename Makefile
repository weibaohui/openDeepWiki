.PHONY: all build run dev clean air

all: build

# Build backend with embedded frontend
build-backend:
	@echo "Building backend with embedded frontend..."
	cd backend && go build -o bin/server ./cmd/server/
	@echo "构建当前平台可执行文件..."
	@mkdir -p backend/bin
	@GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) \
	    CGO_ENABLED=0 go build -ldflags "-s -w  -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT)  -X main.GitTag=$(GIT_TAG)  -X main.GitRepo=$(GIT_REPOSITORY)  -X main.BuildDate=$(BUILD_DATE) -X main.InnerModel=$(MODEL) -X main.InnerApiKey=$(API_KEY) -X main.InnerApiUrl=$(API_URL) " \
	    -o "backend/bin/server" .
# Build frontend
build-frontend:
	@echo "Building frontend..."
	cd frontend && pnpm run build

# Prepare embed directory (copy frontend dist to backend internal embed)
prepare-embed:
	@echo "Preparing embed directory..."
	@mkdir -p backend/internal/embed/ui/dist
	@rm -rf backend/internal/embed/ui/dist/*
	@cp -r frontend/dist/* backend/internal/embed/ui/dist/
	@echo "Frontend files copied to backend/internal/embed/ui/dist/"

# Build all (build frontend, prepare embed, build backend, cleanup)
build: build-frontend prepare-embed build-backend cleanup-embed

# Cleanup embed directory after build
cleanup-embed:
	@echo "Cleaning up embed directory..."
	@rm -rf backend/internal/embed/ui/dist/*
	@touch backend/internal/embed/ui/dist/.keep
	@echo "Embed directory cleaned"

# Run backend
run-backend:
	cd backend && ./bin/server -v 6

# Run frontend dev server
run-frontend:
	cd frontend && pnpm run dev

# Development mode with air (hot reload)
air:
	@echo "Starting backend with air (hot reload)..."
	@echo "Backend: http://localhost:8080"
	@echo "Logs: -v 6 enabled for debugging"
	cd backend && air

# Development mode - run both with air
dev:
	@echo "Starting backend and frontend in development mode..."
	@echo "Backend: http://localhost:8080 (with hot reload)"
	@echo "Frontend: http://localhost:5173"
	@trap 'kill 0' EXIT; \
	cd backend && air & \
	cd frontend && pnpm run dev

# Clean build artifacts
clean:
	rm -rf backend/bin
	rm -rf backend/tmp
	rm -rf backend/internal/embed/ui/dist
	rm -rf frontend/dist

# Setup - install dependencies
setup:
	cd backend && go mod tidy
	cd frontend && pnpm install
	@command -v air >/dev/null 2>&1 || { echo "Installing air..."; go install github.com/air-verse/air@latest; }

# Initialize config
init-config:
	@if [ ! -f backend/config.yaml ]; then \
		cp backend/config.yaml.example backend/config.yaml; \
		echo "Created backend/config.yaml - please update with your API keys"; \
	else \
		echo "backend/config.yaml already exists"; \
	fi
