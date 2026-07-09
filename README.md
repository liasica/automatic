# automatic

基于飞书（Lark）考勤 API 的个人自动化打卡工具。通过轮询 OpenWrt 路由器上的设备在线状态感知「到岗 / 离岗」，自动完成上下班打卡，并内置节假日识别、失败重试、兜底打卡与 Web 考勤仪表盘。

## 功能特性

- **设备感知打卡**：每分钟轮询 OpenWrt 设备列表，按 MAC 地址匹配用户；设备上线触发上班打卡，设备离线触发下班打卡
- **打卡时间随机化**：在配置的分钟范围内随机偏移，并随机化秒数，避免每天打卡时间完全一致
- **节假日识别**：接入 [holiday-cn](https://github.com/NateScarlet/holiday-cn) 年度数据，自动区分工作日 / 周末 / 法定节假日（含调休），数据落盘缓存并每日 00:05 刷新
- **兜底打卡**：工作日 9 点后仍无上班记录、21 点后有上班无下班记录时，自动在预设时间窗口内补卡
- **失败重试**：打卡失败的请求会记录到 Redis，启动时补偿 + 定时重试（默认每 10 分钟）
- **幂等保护**：以「用户 + 日期 + 打卡类型」为键做 Redis 幂等，且打卡前先查询飞书已有记录去重
- **打卡通知**：打卡成功后通过飞书消息通知用户
- **Web Dashboard**：内嵌 Vue 3 前端（`go:embed` 打包进二进制），支持查看 / 手动添加 / 删除打卡记录、统计每日与合计工时、管理加班记录（飞书多维表格）

## 工作原理

```
OpenWrt 设备轮询 ──在线/离线事件──▶ 打卡判定
                                      │
                     ┌────────────────┼─────────────────┐
                     ▼                ▼                 ▼
               幂等键抢占        已有记录去重       节假日判断
                     │                                  │
                     └───────▶ 飞书考勤 API 打卡 ◀──────┘
                                      │
                        成功 ──▶ 飞书消息通知
                        失败 ──▶ Redis 记录 ──▶ 定时重试
```

## 目录结构

```
cmd/automatic/    CLI 入口与子命令（punch、serve）
core/             配置、Redis 缓存、节假日管理
di/               go.uber.org/fx 依赖注入组装
integration/      外部系统对接（feishu 飞书、openwrt 路由器）
service/          Echo HTTP 服务与 REST API handler
dashboard/        Vue 3 + Vite 前端（构建产物嵌入二进制）
docs/             设计 spec 与实施计划
```

## 快速开始

### 依赖

- Go 1.25+
- Redis（幂等键与失败重试队列）
- OpenWrt 路由器（提供设备在线状态接口）
- 飞书自建应用（需具备考勤读写、消息发送权限；加班功能需多维表格权限）
- 前端构建需 Node.js + pnpm

### 配置

配置文件默认路径为 `./configs/config.yaml`（`configs/` 已被 gitignore，需自行创建），也可通过 `--config` 参数或 `CONFIG` 环境变量指定：

```yaml
redis:
  addr: 127.0.0.1:6379
  db: 0

lark:
  appId: cli_xxx
  appSecret: xxx
  overtimeAppToken: xxx        # 加班记录多维表格 app_token（可选）
  redirectURL: ""              # 飞书 OAuth 回调 URL，留空按请求自动推导

openwrt:
  url: http://192.168.1.1      # OpenWrt 设备状态接口地址

retry:
  failedPunchInterval: 10m     # 失败打卡重试间隔，默认 10m

users:
  - id: ou_xxx                 # 飞书用户 userId
    macAddresses:
      - AA:BB:CC:DD:EE:FF      # 用于匹配的设备 MAC 地址
    checkIn:
      latest: "09:00"          # 晚于该时间上线不再触发上班打卡
      from: 0                  # 打卡时间向前随机偏移范围（分钟）
      to: 30
    checkOut:
      earliest: "18:00"        # 早于该时间离线不触发下班打卡
      from: 0                  # 打卡时间向后随机偏移范围（分钟）
      to: 30
```

### 构建与运行

```bash
# 完整构建（前端 + 后端，产物为 build/release/automatic）
make all

# 仅构建后端
make build

# 启动自动打卡（同时在 :9876 启动 Web Dashboard）
automatic punch run

# 禁用 Dashboard 仅打卡
automatic punch run --http-addr=""

# 仅启动 Dashboard（前端联调用，生产请用 punch run）
automatic serve
```

### CLI 命令

```bash
automatic punch run            # 启动自动打卡服务（含 Dashboard）
automatic punch query -u <userId> -f 2026-07-01          # 查询打卡记录
automatic punch add -u <userId> -t "2026-07-01 09:00:00" # 手动添加打卡
automatic punch del --ids <recordId>                     # 删除打卡记录
automatic punch retry-failed   # 立即重试失败的打卡
automatic serve                # 仅启动 Web Dashboard
```

## Docker 部署

仓库提供 `Dockerfile` 与 `docker-compose.yaml`，配置目录挂载到容器 `/app/configs`：

```bash
docker compose up -d
```

## 开发

```bash
# 后端测试
go test ./...

# 前端开发（热更新）
cd dashboard && pnpm install && pnpm dev

# 前端构建（产物 dashboard/dist/ 由 go:embed 打包）
make dashboard
```

前端说明详见 [dashboard/README.md](dashboard/README.md)，设计文档见 `docs/superpowers/`。
