# openDeepWiki

English | [简体中文](./README.md)

## Overview

openDeepWiki is an AI-powered intelligent code repository analysis platform that automatically analyzes any GitHub repository and generates structured project documentation. By combining static code analysis with Large Language Models (LLM), it helps developers quickly understand the architecture, APIs, and business flows of open-source projects.

## 🌟 Try It Online

[https://opendeepwiki.fly.dev/](https://opendeepwiki.fly.dev/)

Experience the powerful features of openDeepWiki immediately without installation or configuration!

## Key Features

- 🚀 **One-Click Analysis**: Enter a GitHub URL to automatically clone and analyze the repository
- 📊 **Intelligent Analysis**: Static analysis + LLM deep analysis for structured documentation
- 📝 **Standardized Output**: Auto-generates 5 types of documents: Overview, Architecture, API, Business Flow, and Deployment
- 🔄 **Task Management**: Visual task progress tracking with support for individual runs, retries, and forced resets
- 📖 **Online Reading**: Built-in Markdown rendering with support for online editing and export
- 🌐 **Multi-Source Support**: Works with public repositories and private repositories (requires GitHub Token)
- 🎨 **Modern UI**: Built with React + Ant Design, supports multiple languages and themes

## Tech Stack

### Backend
- **Language**: Go 1.24+
- **Framework**: Gin
- **Database**: SQLite (default) / MySQL
- **ORM**: GORM
- **Logging**: klog
- **Dev Tool**: Air (hot reload)

### Frontend
- **Framework**: React 19 + TypeScript
- **Build Tool**: Vite
- **UI Library**: Ant Design 6
- **Markdown**: react-markdown / react-md-editor
- **Routing**: React Router 7

### AI Integration
- OpenAI-compatible API support
- Configurable API endpoint, model, and token
- Environment variable configuration support

## Quick Start

### Prerequisites

- Go 1.24+
- Node.js 18+
- Git

### Installation

```bash
# 1. Clone the repository
git clone https://github.com/yourusername/openDeepWiki.git
cd openDeepWiki

# 2. Install dependencies
make setup

# 3. Initialize configuration
make init-config

# 4. Edit config file and set LLM API Key
vim backend/config.yaml
# Or use environment variables
export OPENAI_API_KEY="your-api-key"
export OPENAI_BASE_URL="https://api.openai.com/v1"
export OPENAI_MODEL_NAME="gpt-4o"
```

### Start Services

```bash
# Development mode (recommended): Start both frontend and backend with hot reload
make dev

# Or start separately
make air           # Backend (with hot reload)
make run-frontend  # Frontend

# Production mode
make build
make run-backend
```

### Access URLs

- Frontend: http://localhost:5173
- Backend API: http://localhost:8080

## Usage Guide

### 1. Configure LLM

Configure the LLM API before first use:

**Option 1: Via Configuration File**

Edit `backend/config.yaml`:

```yaml
llm:
  api_url: "https://api.openai.com/v1"
  api_key: "your-api-key"
  model: "gpt-4o"
  max_tokens: 4096
```

**Option 2: Via Environment Variables (Recommended)**

```bash
export OPENAI_API_KEY="your-api-key"
export OPENAI_BASE_URL="https://api.openai.com/v1"
export OPENAI_MODEL_NAME="gpt-4o"
```

**Option 3: Via Web Interface**

Visit `http://localhost:5173/config` to configure.

### 2. Analyze a Repository

1. Enter a GitHub repository URL on the homepage (supports both https and git@ formats)
2. Click "Add" to automatically clone the repository
3. After cloning, click "Run All Tasks" to start analysis
4. Wait for task completion (5 tasks: Overview, Architecture, API, Business Flow, Deployment)
5. Click "View Documentation" to read the generated results

### 3. Document Management

- **Online Reading**: Navigation tree on the left, Markdown rendering on the right
- **Online Editing**: Click the "Edit" button to modify document content
- **Export**: Export individual documents or entire documentation package

## Project Structure

```
openDeepWiki/
├── backend/              # Go backend
│   ├── cmd/server/      # Entry point
│   ├── config/          # Configuration management
│   ├── internal/
│   │   ├── handler/     # HTTP handlers
│   │   ├── model/       # Data models
│   │   ├── repository/  # Data access layer
│   │   ├── service/     # Business logic layer
│   │   │   └── analyzer/ # Analysis engine (static + LLM)
│   │   ├── router/      # Route configuration
│   │   └── pkg/         # Utilities (git/llm/database)
│   ├── go.mod
│   └── config.yaml.example
├── frontend/             # React frontend
│   ├── src/
│   │   ├── components/  # Common components
│   │   ├── pages/       # Page components
│   │   ├── services/    # API calls
│   │   ├── i18n/        # Internationalization
│   │   └── types/       # TypeScript types
│   └── package.json
├── doc/                 # Project documentation
│   ├── 开发规范/
│   └── 需求/
├── Makefile             # Build scripts
└── README.md            # This file
```

## Generated Document Types

Each analyzed repository generates 5 documents:

| Document Name         | Filename         | Description                                             |
| --------------------- | ---------------- | ------------------------------------------------------- |
| Project Overview      | overview.md      | Basic info, tech stack, directory structure             |
| Architecture Analysis | architecture.md  | Overall architecture, module division, dependencies     |
| Core APIs             | api.md           | API interfaces, function signatures, inter-module calls |
| Business Flow         | business-flow.md | Core business logic, data flow                          |
| Deployment Config     | deployment.md    | Configuration files, deployment methods, requirements   |

## Configuration

Complete configuration example (`config.yaml`):

```yaml
server:
  port: "8080"
  mode: "debug"  # debug or release

database:
  type: "sqlite"  # sqlite or mysql
  dsn: "./data/app.db"

llm:
  api_url: "https://api.openai.com/v1"
  api_key: ""  # Recommended to use environment variables
  model: "gpt-4o"
  max_tokens: 4096

github:
  token: ""  # For accessing private repositories

data:
  dir: "./data"
  repo_dir: "./data/repos"
```

## Common Commands

```bash
# Development
make dev              # Dev mode (frontend + backend + hot reload)
make air              # Backend hot reload
make run-frontend     # Frontend dev server

# Build
make build            # Build both frontend and backend
make build-backend    # Build backend only
make build-frontend   # Build frontend only

# Clean
make clean            # Clean build artifacts

# Others
make setup            # Install dependencies
make init-config      # Initialize config file
```

## Development Standards

This project follows strict development standards. See:

- [Backend Standards](./doc/开发规范/后端规范/)
- [Frontend Standards](./doc/开发规范/前端规范/)

## Roadmap

- [ ] Docker containerized deployment
- [ ] User authentication and multi-user support
- [ ] Custom analysis templates
- [ ] More programming language support
- [ ] Batch import and scheduled updates
- [ ] Code change tracking and incremental analysis

## License

[MIT License](./LICENSE)

## Contributing

Issues and Pull Requests are welcome!

## Contact

For questions or suggestions, please submit an Issue.


## Test
dddd
