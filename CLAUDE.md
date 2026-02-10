# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

openDeepWiki is an AI-powered code repository analysis platform that automatically analyzes GitHub repositories and generates structured documentation. It combines static code analysis with Large Language Models (LLM) to help developers quickly understand open-source project architecture, APIs, and business logic.

The system has a clear layered architecture:
- **Frontend**: React 19 + TypeScript with Ant Design 6, Vite build system
- **Backend**: Go 1.24+ with Gin framework, SQLite/MySQL database via GORM
- **AI Integration**: OpenAI-compatible LLM interface with configurable API endpoints

## Common Development Commands

### Setup and Installation
```bash
# Install all dependencies (Go modules + frontend packages + air)
make setup

# Initialize configuration file
make init-config
```

### Development Workflow
```bash
# Start both frontend and backend in development mode (recommended)
make dev

# Start backend only with hot reload (Air)
make air

# Start frontend only
make run-frontend

# Build production version (frontend + embedded backend)
make build
```

### Testing
```bash
# Run Go tests for specific package
cd backend && go test ./internal/service/...

# Run Go tests with verbose output
cd backend && go test -v ./...

# Run frontend linting
cd frontend && npm run lint
```

### Building and Deployment
```bash
# Build for current platform
make build

# Build for Linux cross-platform (arm64/amd64)
make build-linux

# Clean build artifacts
make clean
```

## Code Architecture

### Backend Structure (`backend/`)
- **`cmd/server/`**: Main application entry point and server initialization
- **`internal/router/`**: HTTP routing configuration
- **`internal/handler/`**: HTTP request handlers (API endpoints)
- **`internal/service/`**: Business logic layer with service implementations
  - `orchestrator/`: Task orchestration and workflow management
  - `repository/`: Repository analysis and processing
  - `task/`: Task execution and state management
  - `documentgenerator/`: Document generation services
  - `apianalyzer/`: API analysis services
  - `dbmodelparser/`: Database model parsing
  - `dirmaker/`: Directory structure analysis
- **`internal/repository/`**: Data access layer (GORM repositories)
- **`internal/model/`**: Database models and data structures
- **`internal/pkg/`**: Shared utilities and packages
  - `adkagents/`: AI agent framework and LLM integration
  - `git/`: Git operations wrapper
  - `database/`: Database connection and setup
- **`internal/embed/`**: Embedded frontend assets for single-binary deployment

### Frontend Structure (`frontend/src/`)
- **`pages/`**: Main page components (Home, RepoDetail, DocViewer, Settings, APIKeyManager)
- **`components/`**: Reusable UI components
  - `common/`: Common components (LanguageSwitcher, ThemeSwitcher, GitHubPromoBanner)
  - `markdown/`: Markdown rendering components (MarkdownRender, MermaidRender)
  - `settings/`: Settings-related components (APIKeyList, TaskMonitor)
- **`services/`**: API service layer (`api.ts`)
- **`context/`**: React context providers (AppConfigContext)
- **`providers/`**: Higher-order component providers (ThemeProvider)
- **`types/`**: TypeScript type definitions
- **`i18n/`**: Internationalization support

### Key Configuration Files
- **Backend**: `backend/config.yaml` (LLM API configuration, database settings)
- **Frontend**: `frontend/vite.config.ts`, `frontend/eslint.config.js`
- **Development**: `backend/.air.toml` (hot reload configuration)

## Important Notes for Development

1. **Environment Variables**: LLM configuration can be set via environment variables:
   - `OPENAI_API_KEY`: Your API key
   - `OPENAI_BASE_URL`: API endpoint URL
   - `OPENAI_MODEL_NAME`: Model name to use

2. **Database**: The application uses SQLite by default but supports MySQL. Database migrations are handled automatically.

3. **Testing**: Go tests follow standard Go conventions with `_test.go` files. Frontend uses ESLint for code quality.

4. **Build Process**: The production build embeds the frontend assets into the Go binary, creating a single executable.

5. **Task System**: The core functionality revolves around a task-based system where repositories are analyzed through multiple sequential tasks (overview, architecture, APIs, business flow, deployment).

6. **AI Agent Framework**: The `adkagents` package provides a flexible framework for AI agents with tool calling capabilities, supporting various LLM providers.