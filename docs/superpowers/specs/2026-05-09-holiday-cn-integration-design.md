# 接入 holiday-cn 自动更新节假日数据

**日期**: 2026-05-09
**作者**: liasica + Claude
**状态**: Draft

## 背景

`core/holiday.go` 当前用硬编码 `map[string]bool` 维护中国法定节假日和调休工作日，每年需手工更新。本次发现 2026-05-09（劳动节调休工作日）漏配，导致 `GetDayType()` 返回 `weekend`，下游 [`service/handler/attendance.go:152`](../../service/handler/attendance.go) 的考勤判定出错。

## 目标

- 数据源切换到 [NateScarlet/holiday-cn](https://github.com/NateScarlet/holiday-cn)（CI 每日自动同步国务院通知）
- 进程内每日 00:05 自动刷新今年 + 次年的数据
- 网络失败时优雅降级，不阻断 `GetDayType()` 调用
- `GetDayType()` 签名零变更，调用方无感

## 非目标（YAGNI）

- 不支持自定义假期（如公司额外调休）
- 不引入定时任务框架（cron 库），用原生 ticker 即可
- 不暴露管理 API（重启就重新拉取）
- 不持久化历史变更日志

## 数据源

- URL 模板: `https://raw.githubusercontent.com/NateScarlet/holiday-cn/master/{year}.json`
- 数据 schema:
  ```json
  {
    "$schema": "...",
    "year": 2026,
    "papers": ["..."],
    "days": [
      { "name": "劳动节", "date": "2026-05-09", "isOffDay": false }
    ]
  }
  ```
- 关键字段: `date`（YYYY-MM-DD）+ `isOffDay`（true=放假，false=调休工作日）

## 设计

### 文件结构

```
core/
├── holiday.go              # 改写：薄封装 HolidayManager + 保留兜底硬编码
├── holiday_manager.go      # 新增：加载/更新/降级逻辑
├── holiday_fx.go           # 新增：fx Module
├── holiday_manager_test.go # 新增：单测（fixture）
└── holidays/               # 新增：本地缓存目录（运行时写入）
    └── .gitkeep
```

`holidays/` 目录加 `.gitignore` 排除 JSON。**embed 不预置数据**（B 方案）。

### `HolidayManager` 接口

```go
package core

type HolidayManager interface {
    GetDayType(t time.Time) string
}
```

实现 `holidayManager`：

- 字段：`logger *zap.Logger`、`httpClient *resty.Client`、`cacheDir string`、`mu sync.RWMutex`、`years map[int]*yearData`
- `yearData` 结构：`{ holidays map[string]bool; makeups map[string]bool }`（`holidays` 即 `isOffDay=true` 的日子，`makeups` 即 `isOffDay=false` 的日子）

### `GetDayType` 兜底逻辑（B 方案）

```go
func (m *holidayManager) GetDayType(t time.Time) string {
    ds := t.Format("2006-01-02")
    m.mu.RLock()
    yd, ok := m.years[t.Year()]
    m.mu.RUnlock()
    if ok {
        if yd.holidays[ds] { return DayTypeHoliday }
        if yd.makeups[ds] { return DayTypeWorkday }
    } else {
        // 内存中无该年数据 → 回退硬编码兜底
        if fallbackHolidays[ds] { return DayTypeHoliday }
        if fallbackMakeupWorkdays[ds] { return DayTypeWorkday }
    }
    if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
        return DayTypeWeekend
    }
    return DayTypeWorkday
}
```

### 兜底数据

`holiday.go` 内保留缩减版兜底 map：仅当前年 + 次年。**完整 fix 5月9日漏配后保留至下次手工更新**：

```go
var fallbackHolidays = map[string]bool{ /* 2026 + 2027 法定节假日 */ }
var fallbackMakeupWorkdays = map[string]bool{
    /* 2026 */
    "2026-02-14": true, "2026-02-22": true,
    "2026-04-26": true, "2026-05-09": true, // ← 立即修复
    "2026-09-27": true, "2026-10-10": true,
}
```

### 启动流程

```
fx Lifecycle.OnStart:
  1. 创建 cacheDir（如不存在）
  2. 加载本地缓存文件（若存在）→ years map
  3. 启动 goroutine：
     a. 立即拉一次（今年 + 次年）：成功覆盖 years，失败保留缓存/兜底
     b. 计算下次 00:05 的间隔，time.AfterFunc 触发刷新
     c. 之后每 24h time.Ticker 刷新
```

### 刷新流程

```
对 [year, year+1] 中每个 y:
  GET {url}/{y}.json (timeout 30s, retry 3 次指数退避)
    成功 → 解析 → 原子写入 cacheDir/{y}.json → 更新 years[y]
    失败 → log.Warn(年份, 错误); 保留旧数据
```

### fx Module

```go
var HolidayModule = fx.Module("holiday",
    fx.Provide(NewHolidayManager),
    fx.Invoke(func(*holidayManager) {}), // 强制实例化以触发 Lifecycle
)
```

`service/handler/attendance.go:152` 改为通过依赖注入接收 `core.HolidayManager`，调用 `m.GetDayType(t)` 替换原 `core.GetDayType(t)`。

为兼容性，保留包级 `core.GetDayType(t)` 函数 → 委托给单例 manager（fx 提供）。如果不使用 DI 直接调用就走兜底。**推荐改为 DI 注入**。

## 错误处理

| 场景 | 行为 |
|---|---|
| 启动时 cacheDir 不存在 | 创建（0755） |
| 本地缓存损坏 | log.Warn，跳过该年，继续走网络/兜底 |
| 网络全失败 + 无缓存 | log.Warn，全靠兜底；不影响 service 启动 |
| JSON 字段变更 | 解析失败 → log.Error，保留旧数据 |

## 测试

`holiday_manager_test.go`:

- `TestParseHolidayJSON` — 解析示例 JSON
- `TestGetDayType_FromMemory` — years 已加载，返回正确 DayType
- `TestGetDayType_Fallback` — years 未加载，落到 fallbackXxx
- `TestGetDayType_2026_05_09` — 回归测试：必须返回 `DayTypeWorkday`
- `TestRefresh_NetworkError_KeepsOldData` — mock HTTP server 返回 500，确认旧数据未被清空
- `TestPersistAndLoad` — 写入再读取一致

不打真实网络（`httptest.NewServer` mock）。

## 部署 / 迁移

1. 启动后首次拉取 < 5s 即可补齐数据
2. `cacheDir` 默认 `./holidays/`（与二进制同目录），可通过 config 改
3. 兜底硬编码作为最后保险，每次发布前从 holiday-cn 抄一遍当年/次年（一次性，未来可丢弃）
4. 旧 `nationalHolidays`、`makeupWorkdays` 删除，2025 数据不再维护（用户当前需求是 2026 起）

## 风险

- holiday-cn master 分支结构变更 → 测试用例锁定 schema
- GitHub raw.githubusercontent.com 在国内可能被墙 → 文档建议自建镜像或加 HTTP 代理配置
- 两次刷新之间数据被国务院更新（极少） → 重启或等次日 00:05

## 验证

提交前：

- `go build ./...`
- `go test ./core/...`
- `golangci-lint run --new-from-rev=HEAD~1 --timeout=10m`
- 跑一次 `service/handler/attendance.go` 的相关单测，确认 `2026-05-09` 不再返回 weekend
