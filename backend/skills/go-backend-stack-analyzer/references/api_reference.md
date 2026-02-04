# Go 后端技术栈判定规则表（规则参考）

本文件用于给“Go 后端技术栈识别”提供可复用的判定特征：优先从 `go.mod` 的依赖命中候选，再从 `.go` 代码中的 import/调用点提取证据（文件与行号）。

## 置信度建议（统一口径）

- `0.90+`：入口启动路径/注册路径出现典型调用点（例如 `gin.Default()` + `Run()` / `grpc.NewServer()` + `Serve()`）。
- `0.70~0.89`：明确的初始化或调用点，但不在入口（例如 `internal/server/server.go`）。
- `0.40~0.69`：仅在 `go.mod` 出现依赖，或只在单一文件出现 import。
- `<=0.39`：仅在测试/示例目录出现，或仅出现字符串/注释；默认应降级或忽略。

## 类别与特征（常见）

下面的 “import 特征” 可用于初筛，“代码特征” 用于提置信度与抽取行号证据。

### Web 框架 / 路由

- Gin
  - import 特征：`github.com/gin-gonic/gin`
  - 代码特征：`gin.Default(`、`gin.New(`、`r.Run(`、`router.GET(`/`POST(`)
- Echo
  - import 特征：`github.com/labstack/echo/v4`
  - 代码特征：`echo.New(`、`e.Start(`、`e.GET(`)
- Fiber
  - import 特征：`github.com/gofiber/fiber/v2`
  - 代码特征：`fiber.New(`、`app.Listen(`
- Chi
  - import 特征：`github.com/go-chi/chi/v5`
  - 代码特征：`chi.NewRouter(`、`r.Route(`、`r.Use(`
- Gorilla/mux
  - import 特征：`github.com/gorilla/mux`
  - 代码特征：`mux.NewRouter(`

### API 形态

- REST（net/http 原生）
  - import 特征：`net/http`
  - 代码特征：`http.HandleFunc(`、`http.NewServeMux(`、`http.ListenAndServe(`
- GraphQL（gqlgen）
  - import 特征：`github.com/99designs/gqlgen`
  - 代码特征：`handler.NewDefaultServer(`、`generated.NewExecutableSchema(`
- gRPC
  - import 特征：`google.golang.org/grpc`
  - 代码特征：`grpc.NewServer(`、`Register.*Server(`、`reflection.Register(`
- gRPC-Gateway
  - import 特征：`github.com/grpc-ecosystem/grpc-gateway`
  - 代码特征：`runtime.NewServeMux(`、`Register.*HandlerFromEndpoint(`
- OpenAPI / Swagger
  - import 特征：`github.com/swaggo/gin-swagger`、`github.com/swaggo/echo-swagger`、`github.com/go-swagger/go-swagger`
  - 代码特征：`swaggerFiles`、`ginSwagger.WrapHandler(`

### 数据库 / ORM / 迁移

- GORM
  - import 特征：`gorm.io/gorm`、`gorm.io/driver/mysql`、`gorm.io/driver/sqlite`、`gorm.io/driver/postgres`
  - 代码特征：`gorm.Open(`、`db.AutoMigrate(`
- SQLX
  - import 特征：`github.com/jmoiron/sqlx`
  - 代码特征：`sqlx.Connect(`、`db.Get(`、`db.Select(`
- MongoDB
  - import 特征：`go.mongodb.org/mongo-driver/mongo`
  - 代码特征：`mongo.Connect(`
- Redis
  - import 特征：`github.com/redis/go-redis/v9`（或 `github.com/go-redis/redis/v8`）
  - 代码特征：`redis.NewClient(`、`client.Get(`、`client.Set(`
- 迁移（golang-migrate）
  - import 特征：`github.com/golang-migrate/migrate/v4`
  - 代码特征：`migrate.New(`、`m.Up(`)

### 任务 / 定时 / 工作流

- Cron（robfig/cron）
  - import 特征：`github.com/robfig/cron/v3`
  - 代码特征：`cron.New(`、`c.AddFunc(`
- Asynq（Redis 任务队列）
  - import 特征：`github.com/hibiken/asynq`
  - 代码特征：`asynq.NewServer(`、`client.Enqueue(`
- Temporal
  - import 特征：`go.temporal.io/sdk`
  - 代码特征：`worker.New(`、`workflow.ExecuteActivity(`

### 消息队列

- Kafka（segmentio）
  - import 特征：`github.com/segmentio/kafka-go`
  - 代码特征：`kafka.NewReader(`、`kafka.Writer`
- NATS
  - import 特征：`github.com/nats-io/nats.go`
  - 代码特征：`nats.Connect(`、`nc.Subscribe(`
- RabbitMQ（amqp）
  - import 特征：`github.com/rabbitmq/amqp091-go`（或 `github.com/streadway/amqp`）
  - 代码特征：`amqp.Dial(`、`Channel.Consume(`

### 配置

- Viper
  - import 特征：`github.com/spf13/viper`
  - 代码特征：`viper.SetConfigFile(`、`viper.ReadInConfig(`、`viper.Unmarshal(`
- envconfig
  - import 特征：`github.com/kelseyhightower/envconfig`
  - 代码特征：`envconfig.Process(`

### 鉴权 / 安全

- JWT
  - import 特征：`github.com/golang-jwt/jwt/v5`
  - 代码特征：`jwt.Parse(`、`jwt.NewWithClaims(`
- Casbin（RBAC/ABAC）
  - import 特征：`github.com/casbin/casbin/v2`
  - 代码特征：`casbin.NewEnforcer(`
- CORS
  - import 特征：`github.com/gin-contrib/cors`、`github.com/rs/cors`
  - 代码特征：`cors.New(`、`cors.Default(`

### 可观测性

- 日志（zap/logrus/zerolog/klog）
  - import 特征：`go.uber.org/zap`、`github.com/sirupsen/logrus`、`github.com/rs/zerolog`、`k8s.io/klog/v2`
- OpenTelemetry
  - import 特征：`go.opentelemetry.io/otel`
  - 代码特征：`otel.Tracer(`、`trace.SpanFromContext(`
- Prometheus
  - import 特征：`github.com/prometheus/client_golang/prometheus`
  - 代码特征：`promhttp.Handler(`、`prometheus.NewRegistry(`

### AI / LLM 集成

- Eino（CloudWeGo Eino 框架 / ADK / Compose）
  - import 特征：`github.com/cloudwego/eino`（及其子包如 `.../adk`、`.../compose`、`.../callbacks`、`.../schema`）
  - 代码特征：`adk.NewChatModelAgent(`、`adk.NewSequentialAgent(`、`adk.NewRunner(`、`compose.ToolsNodeConfig`、`callbacks.AppendGlobalHandlers(`
- OpenAI（Eino-Ext ChatModel）
  - import 特征：`github.com/cloudwego/eino-ext/components/model/openai`
  - 代码特征：`openai.NewChatModel(`、`openai.ChatModelConfig{`
- OpenAI 兼容接口（Chat Completions HTTP）
  - 代码特征：`/chat/completions`、`json:"tool_choice,omitempty"`、`json:"tool_calls,omitempty"`、`Role: "tool"`（通常用于 Function Calling / Tool Calling）

### 部署 / 运行时（非 Go 代码证据）

- Docker（容器化）
  - 文件特征：`Dockerfile`、`Dockerfile.*`
  - 内容特征：`FROM`、`COPY --from=`（多阶段构建）、`EXPOSE`、`CMD`、`ENTRYPOINT`
- Docker Compose
  - 文件特征：`docker-compose.yml` / `docker-compose.yaml` / `compose.yml` / `compose.yaml`
  - 内容特征：`services:`、`image:`、`build:`
- Kubernetes
  - 文件特征：常见目录 `k8s/`、`kubernetes/`、`manifests/`、`deploy/` 下的 `*.yaml|*.yml`
  - 内容特征：`apiVersion:`、`kind:`、`metadata:`
- Helm
  - 文件特征：`Chart.yaml`（通常位于 `charts/` 或 `helm/`）
  - 内容特征：`name:`、`version:`、`apiVersion:`
