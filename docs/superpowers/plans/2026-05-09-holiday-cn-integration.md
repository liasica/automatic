# 接入 holiday-cn 自动更新节假日数据 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `core.GetDayType` 的节假日数据源从硬编码切换为运行时从 `NateScarlet/holiday-cn` 拉取，并保留缩减版兜底（含 2026-05-09 hotfix）以应对网络失败。

**Architecture:** 引入 `HolidayManager` 接口，使用 fx Lifecycle 在 OnStart 时加载磁盘缓存并启动后台 goroutine（立即拉取一次 + 每日 00:05 刷新今年与次年）。包级 `core.GetDayType` 改为委托给包级单例 manager；manager 未初始化时回落到 `fallbackHolidays/fallbackMakeupWorkdays`，再不命中则按 `time.Weekday` 判定。这样 `service/handler/attendance.go:152` 等调用方零改动。

**Tech Stack:** Go 1.25 / fx / zap / resty.dev/v3 / `httptest`（测试）。

参考 spec：`docs/superpowers/specs/2026-05-09-holiday-cn-integration-design.md`

---

## 关键决策记录

1. **保留包级 `core.GetDayType(t time.Time) string` API**，调用方零修改（spec 给的是"推荐改为 DI 注入"，本 plan 选最小侵入路径）。
2. **包级单例 manager + sync.RWMutex** 由 `holiday_fx.go` 的 `fx.Invoke` 在 OnStart 时通过 `core.SetHolidayManager(m)` 安装；未安装时（如单元测试或 `core.GetDayType` 早于 fx 启动调用）走兜底 maps。
3. **兜底 maps 只列 2026**，2027 由网络拉取补全；网络失败且无 2027 缓存时，2027 的工作日/周末判定退化为按 `time.Weekday`（与目前行为一致），可接受。
4. **缓存目录用绝对路径或相对二进制目录**，默认 `./holidays/`（不通过 config 暴露，YAGNI）。
5. **HTTP 客户端用 `resty.dev/v3`** 与项目其他 integration 保持一致。
6. **不动 `service/handler/attendance.go`、`cmd/automatic/internal/command/httpserver.go`**：包级函数委托足以覆盖。

---

## 文件结构

| 路径 | 操作 | 责任 |
|------|------|------|
| `core/holiday.go` | 改写 | 常量、缩减兜底 maps、包级 `GetDayType`/`SetHolidayManager` 委托 |
| `core/holiday_manager.go` | 新增 | `HolidayManager` 接口、`holidayManager` 实现（解析、磁盘读写、HTTP 刷新、Lifecycle 钩子） |
| `core/holiday_fx.go` | 新增 | `HolidayModule`（`fx.Provide(NewHolidayManager)` + `fx.Invoke` 触发 Lifecycle 与单例安装） |
| `core/holiday_manager_test.go` | 新增 | parse / persist / GetDayType / refresh 单元测试（含 2026-05-09 回归） |
| `core/holidays/.gitkeep` | 新增 | 占位，使运行时缓存目录在仓库中存在 |
| `.gitignore` | 修改 | 排除 `core/holidays/*.json` |
| `di/di.go` | 修改 | 在 `fx.Provide` 中追加 `core.NewHolidayManager` 与 `fx.Invoke` 触发安装 |

---

## Task 1: 项目骨架与 .gitignore

**Files:**
- Create: `core/holidays/.gitkeep`
- Modify: `.gitignore`

- [ ] **Step 1: 创建缓存目录占位文件**

```bash
mkdir -p core/holidays
touch core/holidays/.gitkeep
```

- [ ] **Step 2: 查看现有 .gitignore**

Run: `cat .gitignore`
Expected: 输出现有忽略规则；如不存在则 `ls .gitignore` 报 "No such file"，本步改为：
```bash
ls .gitignore
```
若不存在则在下一步 `Edit` 改用 `Write` 创建。

- [ ] **Step 3: 在 .gitignore 中追加缓存忽略规则**

如已存在 `.gitignore`，用 Edit 在文件末尾追加：

```
# holiday-cn 运行时缓存（不入库；保留目录与 .gitkeep）
core/holidays/*.json
```

如不存在则用 Write 创建包含上述两行。

- [ ] **Step 4: 校验占位生效**

Run:
```bash
git status --short core/holidays .gitignore
```
Expected: 看到 `?? core/holidays/.gitkeep` 与 `M  .gitignore`（若新建则 `?? .gitignore`）。

- [ ] **Step 5: 提交**

```bash
git add core/holidays/.gitkeep .gitignore
git commit -m "chore(holiday): 准备运行时缓存目录与忽略规则"
```

---

## Task 2: 缩减兜底 maps 与包级 fallback 函数

**Files:**
- Modify: `core/holiday.go`

> 旧 `nationalHolidays`/`makeupWorkdays`（2025+2026）整体替换为 `fallbackHolidays`/`fallbackMakeupWorkdays`（仅 2026），同时把 `GetDayType` 拆出 `fallbackGetDayType` 并暂时让 `GetDayType` 直接调用它（manager 委托在 Task 8 接上，这样本 Task 单独可编译可测）。

- [ ] **Step 1: 重写 `core/holiday.go` 全文**

用 Write 覆写为以下内容（注意：`SetHolidayManager`/`getHolidayManager` 在 Task 8 才填实，本 Task 先放空骨架，方便 Task 5 单测引用 manager 类型）：

```go
// Copyright (C) automatic. 2026-present.
//
// Created at 2026-04-15, by liasica

package core

import (
	"sync/atomic"
	"time"
)

const (
	DayTypeWorkday = "workday" // 工作日
	DayTypeWeekend = "weekend" // 周末
	DayTypeHoliday = "holiday" // 法定节假日
)

// fallbackHolidays 兜底法定节假日（仅当 manager 未注入或缺失对应年份数据时使用）
// 数据源：国务院 2026 年节假日通知（与 holiday-cn 一致）
// 2027 数据待国务院发布后由 holiday-cn 自动覆盖，本兜底不再补
var fallbackHolidays = map[string]bool{
	// ── 2026 ──
	"2026-01-01": true, // 元旦
	"2026-02-15": true, // 春节
	"2026-02-16": true,
	"2026-02-17": true,
	"2026-02-18": true,
	"2026-02-19": true,
	"2026-02-20": true,
	"2026-02-21": true,
	"2026-04-04": true, // 清明节
	"2026-04-05": true,
	"2026-04-06": true,
	"2026-05-01": true, // 劳动节
	"2026-05-02": true,
	"2026-05-03": true,
	"2026-05-04": true,
	"2026-05-05": true,
	"2026-06-19": true, // 端午节
	"2026-06-20": true,
	"2026-06-21": true,
	"2026-10-01": true, // 国庆节+中秋节
	"2026-10-02": true,
	"2026-10-03": true,
	"2026-10-04": true,
	"2026-10-05": true,
	"2026-10-06": true,
	"2026-10-07": true,
	"2026-10-08": true,
}

// fallbackMakeupWorkdays 兜底调休工作日（仅 2026；含本次发现漏配的 2026-05-09）
var fallbackMakeupWorkdays = map[string]bool{
	"2026-02-14": true, // 春节调休
	"2026-02-22": true, // 春节调休
	"2026-04-26": true, // 劳动节调休
	"2026-05-09": true, // 劳动节调休（hotfix：原硬编码漏配）
	"2026-09-27": true, // 国庆调休
	"2026-10-10": true, // 国庆调休
}

// holidayManagerHolder 包级单例 holder；fx 启动时由 SetHolidayManager 安装
var holidayManagerHolder atomic.Pointer[HolidayManager]

// SetHolidayManager 安装 manager 单例（由 fx Lifecycle.OnStart 调用）
// 重复调用会覆盖旧引用；传 nil 会清空（仅测试用）
func SetHolidayManager(m HolidayManager) {
	if m == nil {
		holidayManagerHolder.Store(nil)
		return
	}
	holidayManagerHolder.Store(&m)
}

// getHolidayManager 取当前 manager；未安装返回 nil
func getHolidayManager() HolidayManager {
	p := holidayManagerHolder.Load()
	if p == nil {
		return nil
	}
	return *p
}

// GetDayType 获取日期类型
// 优先使用注入的 HolidayManager（每日刷新真实数据）；未注入或对应年份缺失时回退到包内兜底 maps
func GetDayType(t time.Time) string {
	if m := getHolidayManager(); m != nil {
		return m.GetDayType(t)
	}
	return fallbackGetDayType(t)
}

// fallbackGetDayType 仅依赖包内 fallback maps 的判定逻辑
// manager 未注入或对应年份既无内存数据也无磁盘缓存时使用
func fallbackGetDayType(t time.Time) string {
	ds := t.Format("2006-01-02")
	if fallbackHolidays[ds] {
		return DayTypeHoliday
	}
	if fallbackMakeupWorkdays[ds] {
		return DayTypeWorkday
	}
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return DayTypeWeekend
	}
	return DayTypeWorkday
}
```

- [ ] **Step 2: 编译检查（HolidayManager 接口尚未定义会失败 → 这是预期，下一步先补一个最小占位）**

Run: `go build ./core/...`
Expected: 报错 `undefined: HolidayManager`。说明本 Task 必须在同一 commit 内为 `holiday_manager.go` 留一个最小骨架。继续 Step 3。

- [ ] **Step 3: 创建最小 `core/holiday_manager.go` 骨架**

用 Write 创建（仅声明接口，让 holiday.go 通过编译；具体实现在 Task 3 起逐步填充）：

```go
// Copyright (C) automatic. 2026-present.
//
// Created at 2026-05-09, by liasica

package core

import "time"

// HolidayManager 节假日数据访问接口
// 实现需保证 GetDayType 在并发读取下安全
type HolidayManager interface {
	GetDayType(t time.Time) string
}
```

- [ ] **Step 4: 编译通过验证**

Run: `go build ./core/...`
Expected: 无输出，退出码 0。

- [ ] **Step 5: 写 fallback 与 2026-05-09 回归测试**

用 Write 创建 `core/holiday_manager_test.go`：

```go
// Copyright (C) automatic. 2026-present.
//
// Created at 2026-05-09, by liasica

package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestFallbackGetDayType_2026_05_09 回归测试：劳动节调休工作日不应被识别为周末
func TestFallbackGetDayType_2026_05_09(t *testing.T) {
	day := time.Date(2026, 5, 9, 0, 0, 0, 0, time.Local)
	require.Equal(t, DayTypeWorkday, fallbackGetDayType(day))
}

// TestFallbackGetDayType_Holiday 兜底节假日命中
func TestFallbackGetDayType_Holiday(t *testing.T) {
	day := time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local) // 劳动节
	require.Equal(t, DayTypeHoliday, fallbackGetDayType(day))
}

// TestFallbackGetDayType_Weekend 普通周末
func TestFallbackGetDayType_Weekend(t *testing.T) {
	day := time.Date(2026, 5, 16, 0, 0, 0, 0, time.Local) // 周六，非节假日
	require.Equal(t, DayTypeWeekend, fallbackGetDayType(day))
}

// TestFallbackGetDayType_PlainWorkday 普通工作日
func TestFallbackGetDayType_PlainWorkday(t *testing.T) {
	day := time.Date(2026, 5, 11, 0, 0, 0, 0, time.Local) // 周一
	require.Equal(t, DayTypeWorkday, fallbackGetDayType(day))
}

// TestGetDayType_NoManager_FallsBack manager 未注入时 GetDayType 走 fallback
func TestGetDayType_NoManager_FallsBack(t *testing.T) {
	SetHolidayManager(nil) // 清空（防止其他测试残留）
	day := time.Date(2026, 5, 9, 0, 0, 0, 0, time.Local)
	require.Equal(t, DayTypeWorkday, GetDayType(day))
}
```

- [ ] **Step 6: 跑测试**

Run: `go test -run "TestFallbackGetDayType|TestGetDayType_NoManager" ./core/... -v`
Expected: 5 个用例全部 PASS。

- [ ] **Step 7: 提交**

```bash
git add core/holiday.go core/holiday_manager.go core/holiday_manager_test.go
git commit -m "fix(holiday): 缩减兜底数据并补 2026-05-09 漏配

- 旧 nationalHolidays/makeupWorkdays 改名 fallbackHolidays/fallbackMakeupWorkdays，仅保留 2026
- fallbackMakeupWorkdays 加入 2026-05-09（劳动节调休）
- 引入 HolidayManager 接口与包级单例 holder，未注入时走 fallback 逻辑
- 包级 GetDayType 行为零变更，调用方无感"
```

---

## Task 3: HolidayManager 类型与 JSON 解析

**Files:**
- Modify: `core/holiday_manager.go`
- Modify: `core/holiday_manager_test.go`

- [ ] **Step 1: 写解析测试**

用 Edit 在 `core/holiday_manager_test.go` 文件末尾追加（注意 import 后续可能需要补 `encoding/json`、`os`、`path/filepath` 等，本步只追加测试，编译失败会指出缺哪个 import 再补）：

```go
// TestParseHolidayJSON 验证 holiday-cn 的 JSON schema 解析
func TestParseHolidayJSON(t *testing.T) {
	raw := []byte(`{
		"$schema": "https://example.org/schema",
		"year": 2026,
		"papers": ["paper1"],
		"days": [
			{"name": "劳动节", "date": "2026-05-01", "isOffDay": true},
			{"name": "劳动节", "date": "2026-05-09", "isOffDay": false}
		]
	}`)

	yd, err := parseHolidayJSON(raw)
	require.NoError(t, err)
	require.True(t, yd.holidays["2026-05-01"])
	require.True(t, yd.makeups["2026-05-09"])
	require.False(t, yd.holidays["2026-05-09"])
	require.False(t, yd.makeups["2026-05-01"])
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -run TestParseHolidayJSON ./core/... -v`
Expected: 编译失败 `undefined: parseHolidayJSON` / `undefined: yearData`（本 Task 即将定义）。

- [ ] **Step 3: 在 `core/holiday_manager.go` 实现 yearData + parseHolidayJSON**

用 Edit 把当前 `core/holiday_manager.go` 内容替换为：

```go
// Copyright (C) automatic. 2026-present.
//
// Created at 2026-05-09, by liasica

package core

import (
	"encoding/json"
	"fmt"
	"time"
)

// HolidayManager 节假日数据访问接口
// 实现需保证 GetDayType 在并发读取下安全
type HolidayManager interface {
	GetDayType(t time.Time) string
}

// yearData 单年节假日数据
// holidays: isOffDay=true 的日期集合（法定放假）
// makeups:  isOffDay=false 的日期集合（调休工作日）
type yearData struct {
	holidays map[string]bool
	makeups  map[string]bool
}

// holidayJSON 对齐 holiday-cn 的 JSON schema（仅取必要字段）
type holidayJSON struct {
	Year int `json:"year"`
	Days []struct {
		Name     string `json:"name"`
		Date     string `json:"date"`
		IsOffDay bool   `json:"isOffDay"`
	} `json:"days"`
}

// parseHolidayJSON 解析 holiday-cn 的 JSON，返回该年节假日 / 调休工作日两个集合
func parseHolidayJSON(raw []byte) (*yearData, error) {
	var hj holidayJSON
	if err := json.Unmarshal(raw, &hj); err != nil {
		return nil, fmt.Errorf("解析 holiday JSON 失败: %w", err)
	}

	yd := &yearData{
		holidays: make(map[string]bool, len(hj.Days)),
		makeups:  make(map[string]bool, len(hj.Days)),
	}
	for _, d := range hj.Days {
		if d.Date == "" {
			continue
		}
		if d.IsOffDay {
			yd.holidays[d.Date] = true
		} else {
			yd.makeups[d.Date] = true
		}
	}
	return yd, nil
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test -run TestParseHolidayJSON ./core/... -v`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add core/holiday_manager.go core/holiday_manager_test.go
git commit -m "feat(holiday): 添加 holiday-cn JSON 解析与 yearData 类型"
```

---

## Task 4: 磁盘缓存读写

**Files:**
- Modify: `core/holiday_manager.go`
- Modify: `core/holiday_manager_test.go`

- [ ] **Step 1: 写 persist/load 测试**

在 `core/holiday_manager_test.go` 末尾追加：

```go
// TestPersistAndLoadYear 验证写入再读取数据一致
func TestPersistAndLoadYear(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(`{"year":2026,"days":[
		{"name":"劳动节","date":"2026-05-01","isOffDay":true},
		{"name":"劳动节","date":"2026-05-09","isOffDay":false}
	]}`)

	require.NoError(t, persistYearJSON(dir, 2026, raw))

	yd, err := loadYearFromDisk(dir, 2026)
	require.NoError(t, err)
	require.True(t, yd.holidays["2026-05-01"])
	require.True(t, yd.makeups["2026-05-09"])
}

// TestLoadYear_Missing 文件不存在时返回 nil, nil（让调用方继续走网络/兜底）
func TestLoadYear_Missing(t *testing.T) {
	dir := t.TempDir()
	yd, err := loadYearFromDisk(dir, 2099)
	require.NoError(t, err)
	require.Nil(t, yd)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -run "TestPersistAndLoadYear|TestLoadYear_Missing" ./core/... -v`
Expected: 编译失败 `undefined: persistYearJSON` / `undefined: loadYearFromDisk`。

- [ ] **Step 3: 在 holiday_manager.go 实现读写函数**

用 Edit 在 `holiday_manager.go` 末尾追加：

```go

// holidayCacheFile 拼接缓存文件路径
func holidayCacheFile(dir string, year int) string {
	return filepath.Join(dir, fmt.Sprintf("%d.json", year))
}

// persistYearJSON 原子写入年份缓存（先写 .tmp 再 rename）
func persistYearJSON(dir string, year int, raw []byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建缓存目录失败: %w", err)
	}

	target := holidayCacheFile(dir, year)
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("写入缓存临时文件失败: %w", err)
	}
	if err := os.Rename(tmp, target); err != nil {
		return fmt.Errorf("替换缓存文件失败: %w", err)
	}
	return nil
}

// loadYearFromDisk 读取并解析年份缓存
// 文件不存在时返回 (nil, nil) 让调用方走网络/兜底；解析失败返回错误
func loadYearFromDisk(dir string, year int) (*yearData, error) {
	path := holidayCacheFile(dir, year)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取缓存文件失败: %w", err)
	}
	return parseHolidayJSON(raw)
}
```

同时把 `import` 块更新为：

```go
import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test -run "TestPersistAndLoadYear|TestLoadYear_Missing|TestParseHolidayJSON" ./core/... -v`
Expected: 3 个用例 PASS。

- [ ] **Step 5: 提交**

```bash
git add core/holiday_manager.go core/holiday_manager_test.go
git commit -m "feat(holiday): 添加节假日年份缓存的原子读写"
```

---

## Task 5: HolidayManager.GetDayType 实现

**Files:**
- Modify: `core/holiday_manager.go`
- Modify: `core/holiday_manager_test.go`

- [ ] **Step 1: 写 manager.GetDayType 测试**

在 `core/holiday_manager_test.go` 末尾追加：

```go
// TestManagerGetDayType_FromMemory years 已加载时返回内存数据
func TestManagerGetDayType_FromMemory(t *testing.T) {
	m := newTestManager(t.TempDir())
	m.years[2026] = &yearData{
		holidays: map[string]bool{"2026-05-01": true},
		makeups:  map[string]bool{"2026-05-09": true},
	}

	require.Equal(t, DayTypeHoliday, m.GetDayType(time.Date(2026, 5, 1, 12, 0, 0, 0, time.Local)))
	require.Equal(t, DayTypeWorkday, m.GetDayType(time.Date(2026, 5, 9, 12, 0, 0, 0, time.Local)))
	require.Equal(t, DayTypeWorkday, m.GetDayType(time.Date(2026, 5, 11, 12, 0, 0, 0, time.Local))) // 普通周一
	require.Equal(t, DayTypeWeekend, m.GetDayType(time.Date(2026, 5, 16, 12, 0, 0, 0, time.Local))) // 普通周六
}

// TestManagerGetDayType_Fallback years 未加载该年时回退包级 fallback
func TestManagerGetDayType_Fallback(t *testing.T) {
	m := newTestManager(t.TempDir())
	// 不预置 m.years[2026]，应走 fallback
	require.Equal(t, DayTypeHoliday, m.GetDayType(time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local)))
	require.Equal(t, DayTypeWorkday, m.GetDayType(time.Date(2026, 5, 9, 0, 0, 0, 0, time.Local))) // hotfix 命中
}

// newTestManager 构造一个不启动后台 goroutine 的 manager 用于纯逻辑测试
func newTestManager(cacheDir string) *holidayManager {
	return &holidayManager{
		cacheDir: cacheDir,
		years:    make(map[int]*yearData),
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -run TestManagerGetDayType ./core/... -v`
Expected: 编译失败 `undefined: holidayManager` 等。

- [ ] **Step 3: 在 holiday_manager.go 实现 holidayManager 结构与 GetDayType**

用 Edit 在 `holiday_manager.go` 末尾追加（同时确认 `sync` 已被加入 import）：

```go

// holidayManager 是 HolidayManager 的默认实现
type holidayManager struct {
	cacheDir string
	mu       sync.RWMutex
	years    map[int]*yearData

	// stop 关闭后会终止后台刷新 goroutine
	stop chan struct{}
}

// GetDayType 见 HolidayManager.GetDayType
// 内存中存在该年数据 → 用内存判定；否则 → 包级 fallback
func (m *holidayManager) GetDayType(t time.Time) string {
	ds := t.Format("2006-01-02")
	year := t.Year()

	m.mu.RLock()
	yd, ok := m.years[year]
	m.mu.RUnlock()

	if ok {
		if yd.holidays[ds] {
			return DayTypeHoliday
		}
		if yd.makeups[ds] {
			return DayTypeWorkday
		}
		if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
			return DayTypeWeekend
		}
		return DayTypeWorkday
	}

	return fallbackGetDayType(t)
}

// setYear 在并发安全前提下覆盖单年数据
func (m *holidayManager) setYear(year int, yd *yearData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.years[year] = yd
}
```

把 `import` 块更新为：

```go
import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./core/... -v`
Expected: 之前所有用例 + 本 Task 2 个用例全部 PASS。

- [ ] **Step 5: 提交**

```bash
git add core/holiday_manager.go core/holiday_manager_test.go
git commit -m "feat(holiday): 实现 holidayManager.GetDayType 内存查询与回退"
```

---

## Task 6: HTTP 拉取与回写

**Files:**
- Modify: `core/holiday_manager.go`
- Modify: `core/holiday_manager_test.go`

- [ ] **Step 1: 写 refresh 成功 + 网络失败测试**

在 `core/holiday_manager_test.go` 末尾追加：

```go
// TestRefreshYear_Success 拉取成功时更新内存并写入缓存
func TestRefreshYear_Success(t *testing.T) {
	body := `{"year":2026,"days":[
		{"name":"劳动节","date":"2026-05-01","isOffDay":true},
		{"name":"劳动节","date":"2026-05-09","isOffDay":false}
	]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/2026.json", r.URL.Path)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	dir := t.TempDir()
	m := newTestManager(dir)
	m.urlTemplate = srv.URL + "/%d.json"
	m.httpClient = resty.New().SetTimeout(2 * time.Second)

	require.NoError(t, m.refreshYear(context.Background(), 2026))

	require.Equal(t, DayTypeWorkday, m.GetDayType(time.Date(2026, 5, 9, 0, 0, 0, 0, time.Local)))

	// 已落盘
	yd, err := loadYearFromDisk(dir, 2026)
	require.NoError(t, err)
	require.True(t, yd.makeups["2026-05-09"])
}

// TestRefreshYear_NetworkError_KeepsOldData 网络失败不清空已有内存数据
func TestRefreshYear_NetworkError_KeepsOldData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	m := newTestManager(t.TempDir())
	m.urlTemplate = srv.URL + "/%d.json"
	m.httpClient = resty.New().SetTimeout(500 * time.Millisecond)
	m.years[2026] = &yearData{
		holidays: map[string]bool{},
		makeups:  map[string]bool{"2026-05-09": true},
	}

	err := m.refreshYear(context.Background(), 2026)
	require.Error(t, err)

	// 旧数据仍在
	require.Equal(t, DayTypeWorkday, m.GetDayType(time.Date(2026, 5, 9, 0, 0, 0, 0, time.Local)))
}
```

并把 test 文件 import 块改为：

```go
import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"resty.dev/v3"
)
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -run TestRefreshYear ./core/... -v`
Expected: 编译失败 `m.urlTemplate undefined` / `m.httpClient undefined` / `m.refreshYear undefined`。

- [ ] **Step 3: 在 holiday_manager.go 加字段与 refreshYear 实现**

把 `holidayManager` 结构体改为（用 Edit 在原结构体上扩展）：

```go
// holidayManager 是 HolidayManager 的默认实现
type holidayManager struct {
	urlTemplate string         // 形如 "https://.../%d.json"
	httpClient  *resty.Client
	cacheDir    string
	mu          sync.RWMutex
	years       map[int]*yearData

	// stop 关闭后会终止后台刷新 goroutine
	stop chan struct{}
}
```

在文件末尾追加：

```go

// refreshYear 拉取指定年份数据，成功则更新内存并落盘；失败保留旧数据
func (m *holidayManager) refreshYear(ctx context.Context, year int) error {
	url := fmt.Sprintf(m.urlTemplate, year)
	resp, err := m.httpClient.R().
		SetContext(ctx).
		Get(url)
	if err != nil {
		return fmt.Errorf("拉取 %d 节假日失败: %w", year, err)
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("拉取 %d 节假日返回状态码 %d", year, resp.StatusCode())
	}

	body := resp.Bytes()
	yd, err := parseHolidayJSON(body)
	if err != nil {
		return fmt.Errorf("解析 %d 节假日失败: %w", year, err)
	}

	if err := persistYearJSON(m.cacheDir, year, body); err != nil {
		// 落盘失败仅告警，不影响内存更新
		zap.S().Warnf("写入 %d 节假日缓存失败: %v", year, err)
	}

	m.setYear(year, yd)
	return nil
}
```

把 `import` 更新为：

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"resty.dev/v3"
)
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test -run TestRefreshYear ./core/... -v`
Expected: 2 个用例 PASS。

- [ ] **Step 5: 跑全部 core 测试做回归**

Run: `go test ./core/... -v`
Expected: 全部 PASS。

- [ ] **Step 6: 提交**

```bash
git add core/holiday_manager.go core/holiday_manager_test.go
git commit -m "feat(holiday): 接入 holiday-cn 年度数据拉取与落盘"
```

---

## Task 7: 启动加载 + 每日 00:05 周期刷新

**Files:**
- Modify: `core/holiday_manager.go`

> 该 Task 属于"集成层面"，难以纯单元测试覆盖（涉及 24h ticker 与 time.AfterFunc）。本 Task 不写新单测，依赖 Task 9 端到端验证；但要求实现可被未来注入 clock 替换（YAGNI 原则下，本次只用 `time.Now()`）。

- [ ] **Step 1: 实现 NewHolidayManager 与启动逻辑**

在 `core/holiday_manager.go` 末尾追加：

```go

// holidayURLTemplate 默认数据源 URL 模板
const holidayURLTemplate = "https://raw.githubusercontent.com/NateScarlet/holiday-cn/master/%d.json"

// holidayCacheDir 默认缓存目录（相对工作目录）
const holidayCacheDir = "./holidays"

// NewHolidayManager 构造默认 manager（不启动后台刷新；调用 Start 才启动）
func NewHolidayManager() HolidayManager {
	return &holidayManager{
		urlTemplate: holidayURLTemplate,
		httpClient: resty.New().
			SetTimeout(30 * time.Second).
			SetRetryCount(3).
			SetRetryWaitTime(2 * time.Second).
			SetRetryMaxWaitTime(10 * time.Second),
		cacheDir: holidayCacheDir,
		years:    make(map[int]*yearData),
		stop:     make(chan struct{}),
	}
}

// Start 启动 manager：先加载磁盘缓存，再异步拉取与每日刷新
// 调用方应在 fx Lifecycle.OnStart 内调用
func (m *holidayManager) Start(ctx context.Context) error {
	if err := os.MkdirAll(m.cacheDir, 0o755); err != nil {
		return fmt.Errorf("创建节假日缓存目录失败: %w", err)
	}

	// 启动时把磁盘上 [thisYear, thisYear+1] 的缓存先装进内存
	now := time.Now()
	for _, y := range []int{now.Year(), now.Year() + 1} {
		yd, err := loadYearFromDisk(m.cacheDir, y)
		if err != nil {
			zap.S().Warnf("加载 %d 节假日缓存失败: %v", y, err)
			continue
		}
		if yd != nil {
			m.setYear(y, yd)
		}
	}

	go m.refreshLoop()
	return nil
}

// Stop 停止后台刷新（fx Lifecycle.OnStop 调用）
func (m *holidayManager) Stop(_ context.Context) error {
	select {
	case <-m.stop:
		// 已关闭
	default:
		close(m.stop)
	}
	return nil
}

// refreshLoop 后台 goroutine：立即拉一次 → 等到次日 00:05 → 之后每 24h 触发
func (m *holidayManager) refreshLoop() {
	m.refreshAll()

	for {
		next := nextRefreshTime(time.Now())
		timer := time.NewTimer(time.Until(next))
		select {
		case <-m.stop:
			timer.Stop()
			return
		case <-timer.C:
			m.refreshAll()
		}
	}
}

// refreshAll 刷新今年与次年
func (m *holidayManager) refreshAll() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	now := time.Now()
	for _, y := range []int{now.Year(), now.Year() + 1} {
		if err := m.refreshYear(ctx, y); err != nil {
			zap.S().Warnf("刷新 %d 节假日失败: %v", y, err)
		}
	}
}

// nextRefreshTime 返回下一个 00:05（本地时区）
func nextRefreshTime(now time.Time) time.Time {
	loc := now.Location()
	next := time.Date(now.Year(), now.Month(), now.Day(), 0, 5, 0, 0, loc)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
```

- [ ] **Step 2: 写 nextRefreshTime 单元测试**

在 `core/holiday_manager_test.go` 末尾追加：

```go
// TestNextRefreshTime_BeforeMidnight 当前时间在 00:05 之后，应返回明日 00:05
func TestNextRefreshTime_AfterCutoff(t *testing.T) {
	loc := time.FixedZone("test", 0)
	now := time.Date(2026, 5, 9, 8, 0, 0, 0, loc)
	got := nextRefreshTime(now)
	want := time.Date(2026, 5, 10, 0, 5, 0, 0, loc)
	require.Equal(t, want, got)
}

// TestNextRefreshTime_BeforeCutoff 当前时间在 00:05 之前，应返回今日 00:05
func TestNextRefreshTime_BeforeCutoff(t *testing.T) {
	loc := time.FixedZone("test", 0)
	now := time.Date(2026, 5, 9, 0, 3, 0, 0, loc)
	got := nextRefreshTime(now)
	want := time.Date(2026, 5, 9, 0, 5, 0, 0, loc)
	require.Equal(t, want, got)
}
```

- [ ] **Step 3: 跑测试**

Run: `go test -run TestNextRefreshTime ./core/... -v`
Expected: 2 个用例 PASS。

- [ ] **Step 4: 整体回归**

Run: `go test ./core/... -v`
Expected: 全部 PASS（之前用例 + 本 Task 2 个新用例）。

- [ ] **Step 5: 提交**

```bash
git add core/holiday_manager.go core/holiday_manager_test.go
git commit -m "feat(holiday): 启动加载缓存并按日 00:05 刷新数据"
```

---

## Task 8: fx Module 与单例安装

**Files:**
- Create: `core/holiday_fx.go`

- [ ] **Step 1: 创建 holiday_fx.go**

用 Write 创建：

```go
// Copyright (C) automatic. 2026-present.
//
// Created at 2026-05-09, by liasica

package core

import (
	"context"

	"go.uber.org/fx"
)

// HolidayModule 节假日模块：构造 manager + 在 fx Lifecycle 上挂启停钩子
// 同时把 manager 安装为包级单例，使包级 GetDayType 自动走真实数据
var HolidayModule = fx.Module("holiday",
	fx.Provide(NewHolidayManager),
	fx.Invoke(registerHolidayLifecycle),
)

// registerHolidayLifecycle 把 manager 的 Start/Stop 接到 fx Lifecycle 上
// 同时调用 SetHolidayManager 安装包级单例
func registerHolidayLifecycle(lc fx.Lifecycle, m HolidayManager) {
	hm, ok := m.(*holidayManager)
	if !ok {
		// 测试或自定义实现：仅安装单例，不挂启停
		SetHolidayManager(m)
		return
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := hm.Start(ctx); err != nil {
				return err
			}
			SetHolidayManager(m)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			SetHolidayManager(nil)
			return hm.Stop(ctx)
		},
	})
}
```

- [ ] **Step 2: 编译检查**

Run: `go build ./core/...`
Expected: 无输出，退出码 0。

- [ ] **Step 3: 跑全部 core 测试**

Run: `go test ./core/... -v`
Expected: 全部 PASS。

- [ ] **Step 4: 提交**

```bash
git add core/holiday_fx.go
git commit -m "feat(holiday): 添加 fx Module 与单例 Lifecycle 钩子"
```

---

## Task 9: 接入 di/di.go

**Files:**
- Modify: `di/di.go`

- [ ] **Step 1: 查看当前 di.go**

Run: `cat di/di.go`
Expected: 看到 `fx.New(...)` 调用与现有 Provide 列表。

- [ ] **Step 2: 在 di.go 加入 HolidayModule**

用 Edit 把 `di/di.go` 中：

```go
		// 合并选项
		fx.Options(opts...),
	)
```

替换为：

```go
		// 全局 Modules
		core.HolidayModule,

		// 合并选项
		fx.Options(opts...),
	)
```

- [ ] **Step 3: 编译检查**

Run: `go build ./...`
Expected: 无输出，退出码 0。

- [ ] **Step 4: 跑现有 di 测试，确认 fx 容器仍能正常装配**

Run: `go test ./di/... -v`
Expected: `TestNew` PASS（注意：该测试会真实启动 fx App，从而触发 holiday manager 的 Start。如本机网络不通到 raw.githubusercontent.com，refresh 会 log.Warn 但不会失败 fx 启动 — 这是预期行为）。

> 若 `TestNew` 因网络环境特殊失败：本任务允许在测试中通过 `fx.Decorate` 注入一个无网络副作用的 manager。但默认期望即使无网络，`Start` 也只是 warn 而不报错（refreshYear 错误被 refreshAll 吞掉），所以测试应当通过。

- [ ] **Step 5: 提交**

```bash
git add di/di.go
git commit -m "feat(di): 注入节假日 HolidayModule"
```

---

## Task 10: 端到端验证与 lint

**Files:** （仅运行验证命令，无代码改动）

- [ ] **Step 1: `go build` 全包**

Run: `go build ./...`
Expected: 无输出，退出码 0。

- [ ] **Step 2: `go test` 全包**

Run: `go test ./...`
Expected: 全部包 PASS（如 `service/handler` 等无变更包应保持原状）。

- [ ] **Step 3: 关键回归断言（命令行手测）**

写一个一次性 main，确认 `core.GetDayType` 行为正确。在 `/tmp/holiday_check.go` 写：

```go
package main

import (
	"fmt"
	"time"

	"automatic/core"
)

func main() {
	for _, d := range []time.Time{
		time.Date(2026, 5, 9, 0, 0, 0, 0, time.Local),
		time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local),
		time.Date(2026, 5, 11, 0, 0, 0, 0, time.Local),
		time.Date(2026, 5, 16, 0, 0, 0, 0, time.Local),
	} {
		fmt.Printf("%s -> %s\n", d.Format("2006-01-02"), core.GetDayType(d))
	}
}
```

Run:
```bash
cd /Users/liasica/projects/go/automatic && go run /tmp/holiday_check.go
```

Expected:
```
2026-05-09 -> workday
2026-05-01 -> holiday
2026-05-11 -> workday
2026-05-16 -> weekend
```

跑完删除 `/tmp/holiday_check.go`。

- [ ] **Step 4: golangci-lint 增量检查**

Run:
```bash
/usr/local/bin/gclint run --config .golangci.yml --new-from-rev=HEAD~1 --timeout=10m
```

Expected: `0 issues.`（如有 issue 则按规则修复，每个修复独立 commit）。

> 注：HEAD~1 是相对位置，本次 commit 数量较多，可改为锚点 commit 之前的 master，例如 `--new-from-rev=origin/master`，确保覆盖到本次 plan 全部 commit。

- [ ] **Step 5: 验收 attendance 入口路径未变**

Run: `grep -n "core.GetDayType" service/handler/attendance.go`
Expected: 看到 `attendance.go:152: ... core.GetDayType(t)`，调用代码零修改。

- [ ] **Step 6: 总结 commit 历史**

Run: `git log --oneline origin/master..HEAD`
Expected: 看到本计划产出的 ~9 个 commits（chore + 7 feat + 1 fix）。

- [ ] **Step 7: 推送（按需，由用户决定）**

```bash
# 仅在用户确认后执行：
git push origin master
```

---

## 自审清单（已在写作时完成）

- [x] **Spec 覆盖**：spec 的 7 个目标 / 文件结构 / 错误处理 / 测试用例均映射到 Task 1-10。差异点：spec 设想的 attendance.go DI 改造未做（Task 9 决策保留包级 API），已在"关键决策记录"声明。
- [x] **Placeholder 扫描**：每个步骤含完整代码或可执行命令，无 "TODO/TBD/类似 Task N" 等占位。
- [x] **类型一致性**：`yearData` 字段（`holidays`/`makeups`）、`holidayManager` 字段（`urlTemplate`/`httpClient`/`cacheDir`/`mu`/`years`/`stop`）、函数命名（`parseHolidayJSON`/`persistYearJSON`/`loadYearFromDisk`/`refreshYear`/`refreshAll`/`refreshLoop`/`nextRefreshTime`）在所有 Task 中保持一致。
- [x] **DRY**：fallback 逻辑只写一份（`fallbackGetDayType`），manager 与包级 `GetDayType` 共享。
- [x] **YAGNI**：未引入 cron 库 / config 暴露 / DI 重构 attendance / 暴露管理 API。
- [x] **TDD**：每个有逻辑的 Task 先写失败测试再实现。
- [x] **频繁提交**：10 个 Task ≈ 9 个 commit，每个 commit 自包含可回退。
