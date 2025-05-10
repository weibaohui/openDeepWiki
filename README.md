# DeepWiki (Go Version)

DeepWiki 是一个基于 Go 语言开发的复刻网站，旨在提供一个高效、简洁的 Wiki 平台。此项目灵感来源于原版 DeepWiki，使用 Go 语言重新实现，具有高性能和可扩展性。

## 功能特性
- **轻量级**：使用 Go 语言构建，性能优越，资源占用低。
- **Markdown 支持**：支持使用 Markdown 语法编写和编辑内容。
- **搜索功能**：快速搜索页面内容。
- **用户管理**：支持用户注册、登录和权限管理。
- **RESTful API**：提供 API 接口，方便与其他系统集成。

## 技术栈
- **后端**：Go
- **数据库**：SQLite 或 MySQL（可选）
- **前端**：HTML、CSS、JavaScript（可选框架：React 或 Vue）
- **其他**：支持 Docker 部署

## 快速开始

### 环境要求
- Go 1.20 或更高版本
- SQLite 或 MySQL 数据库
- Git

### 安装步骤
1. 克隆项目：
   ```bash
   git clone https://github.com/your-repo/openDeepWiki.git
   cd openDeepWiki
   ```

2. 安装依赖：
   ```bash
   go mod tidy
   ```

3. 配置环境变量：
   创建一个 `.env` 文件并配置以下内容：
   ```env
   DB_TYPE=sqlite  # 或 mysql
   DB_CONNECTION=your_database_connection_string
   PORT=8080
   ```

4. 运行项目：
   ```bash
   go run main.go
   ```

5. 打开浏览器访问：
   ```
   http://localhost:8080
   ```

## 目录结构
```
openDeepWiki/
├── main.go          # 主程序入口
├── README.md        # 项目说明文件
├── go.mod           # Go 模块文件
├── go.sum           # 依赖锁定文件
├── internal/        # 内部模块
│   ├── handlers/    # HTTP 处理程序
│   ├── models/      # 数据模型
│   └── utils/       # 工具函数
├── public/          # 静态文件
├── templates/       # HTML 模板
└── .env.example     # 环境变量示例文件
```

## 贡献
欢迎对本项目提出建议或贡献代码！请提交 Pull Request 或创建 Issue。

## 许可证
本项目基于 MIT 许可证开源。详情请参阅 [LICENSE](LICENSE) 文件。
