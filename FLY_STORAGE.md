# Fly.io 持久化存储配置说明

## 概述

本项目已配置 Fly.io 的持久化存储功能,确保 SQLite 数据库在实例重启后不会丢失。仓库数据不进行持久化存储。

## 存储配置

### fly.toml 配置

```toml
[mounts]
  source = "db_volume"
  destination = "/data/db"
```

- **source**: 存储卷名称
- **destination**: 容器内的挂载路径

### 环境变量

以下环境变量可用于配置应用的数据存储路径:

| 环境变量 | 说明 | 默认值 | Fly.io 建议值 |
|---------|------|--------|--------------|
| `DB_DSN` | SQLite 数据库路径 | `./data/app.db` | `/data/db/app.db` |
| `DATA_DIR` | 数据根目录(仅用于本地开发) | `./data` | 不设置 |
| `REPO_DIR` | 仓库存储目录(不持久化) | `./data/repos` | 不设置 |

## 部署到 Fly.io

### 1. 设置环境变量

在 Fly.io 上设置数据库路径环境变量:

```bash
flyctl secrets set DB_DSN=/data/db/app.db
```

### 2. 创建存储卷

首次部署时,Fly.io 会自动创建存储卷。如果需要手动创建:

```bash
flyctl volumes create db_volume --region sin --size 1
```

### 3. 部署应用

```bash
flyctl deploy
```

## 本地开发

在本地开发时,使用默认值即可:

- `DB_DSN`: `./data/app.db`
- `DATA_DIR`: `./data`
- `REPO_DIR`: `./data/repos`

## 数据迁移

如果需要从本地迁移数据库数据到 Fly.io:

1. 导出本地数据库
2. 使用 `flyctl sftp` 上传到 `/data/db` 目录
3. 重启应用

## 监控

查看存储卷使用情况:

```bash
flyctl volumes list
```

查看具体存储卷详情:

```bash
flyctl volumes show db_volume
```

## 注意事项

1. **存储大小**: 默认存储卷大小为 1GB,可根据需要调整
2. **区域**: 存储卷与应用实例必须在同一区域
3. **备份**: 建议定期备份数据库
4. **性能**: 持久化存储比本地存储慢,不适合高频读写场景
5. **仓库数据**: 仓库数据不进行持久化存储,实例重启后需要重新克隆
