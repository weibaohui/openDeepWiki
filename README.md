# DeepWiki (Go Version)

DeepWiki 是一个基于 Go 语言开发的复刻网站，旨在提供一个高效、简洁的 Wiki 平台。此项目灵感来源于原版 DeepWiki，使用 Go 语言重新实现，具有高性能和可扩展性。

## 功能特性
- **轻量级**：使用 Go 语言构建，性能优越，资源占用低。
- **Markdown 支持**：支持使用 Markdown 语法编写和编辑内容。
- **搜索功能**：快速搜索页面内容。
- **用户管理**：支持用户注册、登录和权限管理。
- **MCP 协议支持**：提供 MCP（Model Context Protocol）接口，供外部工具访问。
- **Docker 支持**：可通过 Docker 一键部署。
- **多数据库支持**：支持 SQLite（默认）和 MySQL。

## 技术栈
- **后端**：Go 1.20+
- **数据库**：SQLite（默认，文件位于 `data/openDeepWiki.db`）或 MySQL
- **前端**：Vite + React（`ui/` 目录，支持热更新）
- **构建工具**：Makefile
- **容器化**：Dockerfile

## 快速开始

### 环境要求
- Go 1.20 或更高版本
- SQLite 或 MySQL 数据库
- Node.js 16+（如需开发/构建前端）
- Git
- Docker（可选）

### 安装步骤
1. 克隆项目：
   ```bash
   git clone https://github.com/your-repo/openDeepWiki.git
   cd openDeepWiki
   ```

2. 安装后端依赖：
   ```bash
   go mod tidy
   ```

3. 启动后端服务：
   ```bash
   go run main.go
   # 或使用 Makefile
   make run
   ```

4. 启动前端（可选）：
   ```bash
   cd ui
   pnpm install  # 或 npm install
   pnpm dev      # 或 npm run dev
   # 访问 http://localhost:3000
   ```
 
## 目录结构
```
openDeepWiki/
├── main.go          # 主程序入口
├── README.md        # 项目说明文件
├── go.mod           # Go 模块文件
├── go.sum           # 依赖锁定文件
├── Makefile         # 构建与运行脚本
├── Dockerfile       # Docker 镜像构建文件
├── bin/             # 可执行文件输出目录
├── data/            # 数据文件（如 SQLite 数据库）
├── internal/        # 内部模块
│   ├── handlers/    # HTTP 处理程序
│   ├── models/      # 数据模型
│   └── utils/       # 工具函数
├── pkg/             # 公共包与业务逻辑
├── public/          # 静态文件
├── templates/       # HTML 模板
├── ui/              # 前端源码（Vite+React）
└── .env.example     # 环境变量示例文件
```

## 贡献
欢迎对本项目提出建议或贡献代码！请提交 Pull Request 或创建 Issue。

## 许可证
本项目基于 MIT 许可证开源。详情请参阅 [LICENSE](LICENSE) 文件。
