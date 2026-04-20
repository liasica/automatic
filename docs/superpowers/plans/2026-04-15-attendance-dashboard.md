# 打卡记录 Dashboard 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为自动打卡系统添加 Web Dashboard，使用 Composio 暗色风格展示每日打卡记录，支持日期筛选、添加和删除操作

**Architecture:** Go 后端新增 Echo HTTP 服务器，提供 REST API 代理飞书考勤 SDK 操作；Vue 3 前端使用 Tailwind CSS 4 实现 Composio 设计风格的单页面应用。开发时前后端分离（Vite dev server + API 代理），生产环境将前端静态文件嵌入 Go 二进制

**Tech Stack:** Go (Echo v4), Vue 3 (Composition API + TypeScript), Tailwind CSS 4, Vite 8

---

## 文件结构

### 后端新增/修改

| 文件 | 职责 |
|------|------|
| `service/server.go` | Echo 服务器初始化、路由注册、中间件 |
| `service/handler/attendance.go` | 打卡记录 API 处理函数（查询/添加/删除）|
| `cmd/automatic/internal/command/serve.go` | `serve` CLI 命令，启动 HTTP 服务器 |
| `cmd/automatic/main.go` | 注册 `serve` 命令 |

### 前端新增/修改

| 文件 | 职责 |
|------|------|
| `dashboard/src/assets/main.css` | Composio 主题变量 + Tailwind 基础样式 |
| `dashboard/src/App.vue` | 主页面布局：标题栏 + 内容区 |
| `dashboard/src/components/DateFilter.vue` | 日期范围筛选组件 |
| `dashboard/src/components/RecordTable.vue` | 打卡记录表格组件 |
| `dashboard/src/components/AddRecordModal.vue` | 添加打卡记录弹窗 |
| `dashboard/src/components/DeleteConfirmModal.vue` | 删除确认弹窗 |
| `dashboard/src/composables/useAttendance.ts` | API 调用 + 状态管理 |
| `dashboard/src/types.ts` | TypeScript 类型定义 |
| `dashboard/vite.config.ts` | 添加 API 代理配置 |
| `dashboard/index.html` | 更新页面标题、引入字体 |

---

### Task 1: 后端 - 添加 Echo 依赖

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: 安装 Echo v4**

```bash
cd /Users/liasica/projects/go/automatic && go get github.com/labstack/echo/v4
```

- [ ] **Step 2: 验证依赖安装**

```bash
grep echo go.mod
```

Expected: `github.com/labstack/echo/v4 v4.x.x`

- [ ] **Step 3: 提交**

```bash
git add go.mod go.sum
git commit -m "feat: 添加 Echo v4 HTTP 框架依赖"
```

---

### Task 2: 后端 - 创建 HTTP 服务器和路由

**Files:**
- Create: `service/server.go`

- [ ] **Step 1: 创建 server.go**

```go
package service

import (
	"context"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"

	"automatic/integration/feishu"
	"automatic/service/handler"
)

type Server struct {
	echo *echo.Echo
}

// NewServer 创建 HTTP 服务器并注册路由
func NewServer(fs *feishu.Feishu) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// 中间件
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type"},
	}))

	// 注册路由
	h := handler.NewAttendance(fs)
	api := e.Group("/api")
	attendance := api.Group("/attendance")
	attendance.GET("/records", h.Query)
	attendance.POST("/records", h.Add)
	attendance.DELETE("/records", h.Delete)

	return &Server{echo: e}
}

// Start 启动 HTTP 服务器
func (s *Server) Start(addr string) error {
	zap.S().Infof("HTTP 服务器启动，监听地址: %s", addr)
	return s.echo.Start(addr)
}

// Shutdown 优雅关闭 HTTP 服务器
func (s *Server) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}
```

- [ ] **Step 2: 验证编译**

```bash
cd /Users/liasica/projects/go/automatic && go build ./service/...
```

注意：此时还没有 handler 包，预期编译失败，进入 Task 3 后解决

---

### Task 3: 后端 - 创建打卡记录 API 处理函数

**Files:**
- Create: `service/handler/attendance.go`

- [ ] **Step 1: 创建 handler 目录和文件**

```go
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"automatic/integration/feishu"
)

type Attendance struct {
	feishu *feishu.Feishu
}

func NewAttendance(fs *feishu.Feishu) *Attendance {
	return &Attendance{feishu: fs}
}

// 查询请求参数
type queryRequest struct {
	UserID string `query:"user_id"`
	From   string `query:"from"`
	To     string `query:"to"`
}

// 打卡记录响应
type record struct {
	RecordID  string `json:"record_id"`
	UserID    string `json:"user_id"`
	CheckTime string `json:"check_time"`
	Timestamp int64  `json:"timestamp"`
}

// Query 查询打卡记录
func (a *Attendance) Query(c echo.Context) error {
	var req queryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "参数错误"})
	}
	if req.UserID == "" || req.From == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user_id 和 from 参数必填"})
	}

	fromTime, err := time.ParseInLocation(time.DateOnly, req.From, time.Local)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "from 日期格式错误，需要 YYYY-MM-DD"})
	}

	var toTime time.Time
	if req.To == "" {
		// 默认查询当天
		toTime = fromTime.Add(24*time.Hour - time.Second)
	} else {
		toTime, err = time.ParseInLocation(time.DateOnly, req.To, time.Local)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "to 日期格式错误，需要 YYYY-MM-DD"})
		}
		toTime = toTime.Add(24*time.Hour - time.Second)
	}

	data, err := a.feishu.UserFlowsQuery(req.UserID, fromTime, toTime)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var records []record
	if data != nil {
		for _, flow := range data.UserFlowResults {
			if flow == nil || flow.CheckTime == nil || flow.RecordId == nil {
				continue
			}
			ts, parseErr := strconv.ParseInt(*flow.CheckTime, 10, 64)
			if parseErr != nil {
				continue
			}
			r := record{
				RecordID:  *flow.RecordId,
				UserID:    *flow.UserId,
				CheckTime: time.Unix(ts, 0).Format(time.DateTime),
				Timestamp: ts,
			}
			records = append(records, r)
		}
	}

	if records == nil {
		records = []record{}
	}

	return c.JSON(http.StatusOK, map[string]any{"records": records})
}

// 添加请求
type addRequest struct {
	UserID    string `json:"user_id"`
	CheckTime string `json:"check_time"`
}

// Add 添加打卡记录
func (a *Attendance) Add(c echo.Context) error {
	var req addRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "参数错误"})
	}
	if req.UserID == "" || req.CheckTime == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user_id 和 check_time 参数必填"})
	}

	checkTime, err := time.ParseInLocation(time.DateTime, req.CheckTime, time.Local)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "check_time 格式错误，需要 YYYY-MM-DD HH:MM:SS"})
	}

	err = a.feishu.UserFlowsCreate(req.UserID, checkTime)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "添加成功"})
}

// 删除请求
type deleteRequest struct {
	RecordIDs []string `json:"record_ids"`
}

// Delete 删除打卡记录
func (a *Attendance) Delete(c echo.Context) error {
	var req deleteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "参数错误"})
	}
	if len(req.RecordIDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "record_ids 不能为空"})
	}

	successIDs, failIDs, err := a.feishu.UserFlowsDelete(req.RecordIDs...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success_ids": successIDs,
		"fail_ids":    failIDs,
	})
}
```

- [ ] **Step 2: 验证编译**

```bash
cd /Users/liasica/projects/go/automatic && go build ./...
```

Expected: 编译成功

- [ ] **Step 3: 提交**

```bash
git add service/
git commit -m "feat: 添加打卡记录 HTTP API (查询/添加/删除)"
```

---

### Task 4: 后端 - 创建 serve CLI 命令

**Files:**
- Create: `cmd/automatic/internal/command/serve.go`
- Modify: `cmd/automatic/main.go`

- [ ] **Step 1: 创建 serve.go**

```go
package command

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"automatic/core"
	"automatic/di"
	"automatic/integration/feishu"
	"automatic/service"
)

type Serve struct {
	*cli.Command
}

func NewServe() *Serve {
	s := &Serve{}
	s.Command = &cli.Command{
		Name:        "serve",
		Usage:       "启动 Web Dashboard 服务",
		Description: "启动 HTTP 服务器，提供打卡记录管理 Web 界面和 API",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "addr",
				Aliases: []string{"a"},
				Value:   ":8080",
				Usage:   "HTTP 监听地址，例如: :8080",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			addr := cmd.String("addr")
			var srv *service.Server

			app := di.New(cmd, fx.Invoke(func(lc fx.Lifecycle, _ *core.Config, fs *feishu.Feishu) {
				srv = service.NewServer(fs)
				lc.Append(fx.Hook{
					OnStart: func(context.Context) error {
						go func() {
							if err := srv.Start(addr); err != nil {
								zap.S().Infof("HTTP 服务器已关闭: %v", err)
							}
						}()
						return nil
					},
					OnStop: func(ctx context.Context) error {
						return srv.Shutdown(ctx)
					},
				})
			}))

			if err := app.Start(ctx); err != nil {
				return err
			}

			ch := make(chan os.Signal, 1)
			signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(ch)

			select {
			case <-ctx.Done():
			case sig := <-ch:
				zap.S().Infof("收到退出信号: %s", sig.String())
			}

			stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return app.Stop(stopCtx)
		},
	}
	return s
}
```

- [ ] **Step 2: 修改 main.go 注册 serve 命令**

在 `cmd/automatic/main.go` 的 Commands 数组中添加：

```go
Commands: []*cli.Command{
    addGlobalFlags(command.NewPunch().Command),
    addGlobalFlags(command.NewServe().Command),
},
```

- [ ] **Step 3: 验证编译**

```bash
cd /Users/liasica/projects/go/automatic && go build ./cmd/automatic/...
```

Expected: 编译成功

- [ ] **Step 4: 提交**

```bash
git add cmd/automatic/ service/
git commit -m "feat: 添加 serve 命令启动 Web Dashboard HTTP 服务"
```

---

### Task 5: 前端 - TypeScript 类型和 API 层

**Files:**
- Create: `dashboard/src/types.ts`
- Create: `dashboard/src/composables/useAttendance.ts`

- [ ] **Step 1: 创建类型定义**

`dashboard/src/types.ts`:
```typescript
export interface AttendanceRecord {
  record_id: string
  user_id: string
  check_time: string
  timestamp: number
}

export interface QueryParams {
  user_id: string
  from: string
  to?: string
}

export interface AddRecordParams {
  user_id: string
  check_time: string
}
```

- [ ] **Step 2: 创建 API composable**

`dashboard/src/composables/useAttendance.ts`:
```typescript
import { ref } from 'vue'
import type { AttendanceRecord, QueryParams, AddRecordParams } from '@/types'

const API_BASE = '/api/attendance'

export function useAttendance() {
  const records = ref<AttendanceRecord[]>([])
  const loading = ref(false)
  const error = ref('')

  async function queryRecords(params: QueryParams) {
    loading.value = true
    error.value = ''
    try {
      const query = new URLSearchParams({ user_id: params.user_id, from: params.from })
      if (params.to) query.set('to', params.to)
      const resp = await fetch(`${API_BASE}/records?${query}`)
      const data = await resp.json()
      if (!resp.ok) throw new Error(data.error || '查询失败')
      records.value = data.records
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function addRecord(params: AddRecordParams) {
    loading.value = true
    error.value = ''
    try {
      const resp = await fetch(`${API_BASE}/records`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(params),
      })
      const data = await resp.json()
      if (!resp.ok) throw new Error(data.error || '添加失败')
      return true
    } catch (e: any) {
      error.value = e.message
      return false
    } finally {
      loading.value = false
    }
  }

  async function deleteRecords(recordIds: string[]) {
    loading.value = true
    error.value = ''
    try {
      const resp = await fetch(`${API_BASE}/records`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ record_ids: recordIds }),
      })
      const data = await resp.json()
      if (!resp.ok) throw new Error(data.error || '删除失败')
      return { successIds: data.success_ids, failIds: data.fail_ids }
    } catch (e: any) {
      error.value = e.message
      return null
    } finally {
      loading.value = false
    }
  }

  return { records, loading, error, queryRecords, addRecord, deleteRecords }
}
```

- [ ] **Step 3: 提交**

```bash
git add dashboard/src/types.ts dashboard/src/composables/
git commit -m "feat: 添加前端 TypeScript 类型定义和 API 调用层"
```

---

### Task 6: 前端 - Composio 主题和基础样式

**Files:**
- Modify: `dashboard/src/assets/main.css`
- Modify: `dashboard/index.html`
- Modify: `dashboard/vite.config.ts`

- [ ] **Step 1: 更新 index.html**

添加 JetBrains Mono 字体引入，更新标题

- [ ] **Step 2: 更新 main.css**

应用 Composio 设计系统的 CSS 变量和 Tailwind 主题：
- Void Black 背景 (#0f0f0f)
- 白色透明度边框体系
- JetBrains Mono 等宽字体
- Electric Cyan 和 Composio Cobalt 强调色

- [ ] **Step 3: 更新 vite.config.ts**

添加开发环境 API 代理配置到 Go 后端 `:8080`

- [ ] **Step 4: 提交**

```bash
git add dashboard/
git commit -m "feat: 应用 Composio 暗色主题和开发代理配置"
```

---

### Task 7: 前端 - 核心页面和组件

**Files:**
- Modify: `dashboard/src/App.vue`
- Create: `dashboard/src/components/DateFilter.vue`
- Create: `dashboard/src/components/RecordTable.vue`
- Create: `dashboard/src/components/AddRecordModal.vue`
- Create: `dashboard/src/components/DeleteConfirmModal.vue`

- [ ] **Step 1: 创建 DateFilter 组件**

日期范围选择器，包含开始日期、结束日期输入框和查询按钮

- [ ] **Step 2: 创建 RecordTable 组件**

打卡记录表格，显示记录 ID、打卡时间，每行有删除按钮

- [ ] **Step 3: 创建 AddRecordModal 组件**

添加打卡记录弹窗，包含用户 ID 和日期时间选择

- [ ] **Step 4: 创建 DeleteConfirmModal 组件**

删除确认弹窗

- [ ] **Step 5: 更新 App.vue**

组装所有组件：顶部标题栏 + 筛选区 + 记录表格 + 弹窗

- [ ] **Step 6: 验证前端构建**

```bash
cd /Users/liasica/projects/go/automatic/dashboard && pnpm run build
```

- [ ] **Step 7: 提交**

```bash
git add dashboard/src/
git commit -m "feat: 实现打卡记录 Dashboard 页面 (Composio 风格)"
```

---

### Task 8: 集成验证

- [ ] **Step 1: 验证后端编译**

```bash
cd /Users/liasica/projects/go/automatic && go build ./...
```

- [ ] **Step 2: 验证前端构建**

```bash
cd /Users/liasica/projects/go/automatic/dashboard && pnpm run build
```

- [ ] **Step 3: 验证前端 lint**

```bash
cd /Users/liasica/projects/go/automatic/dashboard && pnpm run type-check
```
