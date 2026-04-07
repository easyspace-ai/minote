# Notex 后端容器化与本地开发说明

本文说明如何将 **`cmd/notex`**（统一 Notex API + LangGraph）通过 Docker Compose 运行，以及在开发阶段如何快速重建或热重载。

---

## 一、涉及文件

| 路径 | 说明 |
|------|------|
| `docker/notex/Dockerfile` | 生产向多阶段镜像：编译 `cmd/notex`，默认内置 `config.example.yaml` 为 `/app/config.yaml` |
| `docker/notex/Dockerfile.dev` | 开发镜像：安装 **Air**，配合挂载源码做热重载 |
| `docker/notex/air.toml` | Air 配置：监听 `cmd` / `internal` / `pkg` 下 Go 变更并执行 `go build` |
| 根目录 `docker-compose.yml` | 新增服务 **`notex`**（profile `app`）、**`notex-dev`**（profile `dev`） |
| 根目录 `Makefile` | 常用命令封装（`up-app`、`dev-notex`、`notex-rebuild`、`watch-notex` 等） |

---

## 二、Compose 中的两个 Notex 形态

### 1. `notex`（profile：`app`）

- 使用 **`docker/notex/Dockerfile`** 构建的镜像。
- 适合：**部署验证、与线上一致的运行方式**。
- 容器内会**覆盖**你本机 `.env` 里指向 `localhost` 的数据库 / MarkItDown 等地址，改为 **Compose 服务名**（与在宿主机 `go run` 时的配置可以并存，互不影响）：
  - `POSTGRES_URL` → `postgres:5432`
  - `MARKITDOWN_URL` → `http://markitdown:8787`
  - `DOCREADER_ADDR` → `docreader:50051`
  - `REDIS_ADDR` → `redis:6379`
  - Milvus 健康检查等亦指向 compose 网络内地址  
- **API Key、模型名**等仍从 **`.env`**（`env_file`）读取。
- 挂载：**`./config.yaml` → `/app/config.yaml`（只读）**；数据目录卷 **`youmind-notex-data` → `/data/notex`**。
- 默认映射：**宿主机 `NOTEX_HOST_PORT`（默认 8787）→ 容器 8787**。
- 配置了 **`develop.watch`**：支持 `docker compose watch notex`，在修改 `go.mod` / `go.sum` / `cmd` / `internal` / `pkg` 时**自动重建镜像**（需较新的 Docker Compose）。

### 2. `notex-dev`（profile：`dev`）

- 使用 **`docker/notex/Dockerfile.dev`**，进程为 **Air**。
- 挂载 **整个仓库到 `/src`**，保存 `.go` 后自动编译并重启进程，**无需每次手动 `docker compose build`**。
- 默认映射：**`NOTEX_DEV_HOST_PORT`（默认 8797）→ 容器 8787**，避免与本机 `go run cmd/notex` 默认的 **8787** 冲突。
- 环境变量与依赖关系与 `notex` 一致（同样走 compose 内 Postgres / MarkItDown 等）。

---

## 三、Makefile 常用命令

| 命令 | 作用 |
|------|------|
| `make infra` | 只启动基础设施（Postgres、Redis、MarkItDown、docreader、MinIO 等），**不含** Notex |
| `make up-app` | 启动基础设施 + **`notex`**（需仓库根目录存在 **`config.yaml`**） |
| `make dev-notex` | 尽量停掉容器 **`notex`**，再启动 **`notex-dev`**（Air 热重载） |
| `make notex-build` | 仅构建 Notex 生产镜像 |
| `make notex-rebuild` | **`--build --force-recreate`** 重建并拉起 `notex` |
| `make notex-restart` | 不重新 build，仅 **`--force-recreate`** `notex` |
| `make notex-logs` | 跟踪 `notex` 日志 |
| `make watch-notex` | 对 `notex` 执行 **`docker compose watch`**（改代码触发镜像重建） |
| `make down` | `docker compose down` |

---

## 四、推荐工作流（对齐「WeKnora 式」开发习惯）

1. **只跑依赖、应用在宿主机**（与 `WeKnora/docker-compose.dev.yml` 思路类似）  
   - `make infra` 或 `docker compose up -d`  
   - 本机：`go run ./cmd/notex/main.go`，`.env` 里用 **`localhost` + 映射端口**（如 `POSTGRES_URL`、`MARKITDOWN_URL`）。

2. **应用在容器里、改代码要省事**  
   - 先 `make infra`，再 **`make dev-notex`**，访问 **`http://localhost:8797`**（或你设置的 `NOTEX_DEV_HOST_PORT`）。

3. **验证生产镜像**  
   - `make up-app`，访问 **`http://localhost:8787`**（或 `NOTEX_HOST_PORT`）。

4. **不装 Air、但想少敲 build**  
   - `make up-app` 后，另开终端：`make watch-notex`，改 Go 代码会触发镜像重建（时间比 Air 长，但无需本机 Go 环境参与运行）。

---

## 五、环境与配置注意点

- 使用 **`make up-app` / `dev-notex`** 前，请确保存在 **`config.yaml`**；没有可复制：  
  `cp config.example.yaml config.yaml`
- Compose 中 **`notex` / `notex-dev`** 使用 **`env_file: .env`**；若本机无 `.env`，请先创建（可参考 `.env.example`）。
- 根目录 **`.dockerignore`** 已排除 `web/node_modules`、`three/` 等，以加快 **`docker build`**。
- 更多环境变量说明见 **`.env.example** 中「Notex in Docker Compose」一节。

---

## 六、等价 Docker 命令参考

```bash
# 仅基础设施
docker compose up -d

# 生产 Notex 镜像
docker compose --profile app up -d

# 开发 Notex（Air）
docker compose --profile dev up -d notex-dev
```

---

*文档生成自当前仓库实现；若 Compose/Makefile 有变更，请同步更新本文。*
