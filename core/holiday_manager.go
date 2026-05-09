// Copyright (C) automatic. 2026-present.
//
// Created at 2026-05-09, by liasica

package core

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

// holidayManager 是 HolidayManager 的默认实现
type holidayManager struct {
	urlTemplate string // 形如 "https://.../%d.json"
	httpClient  *resty.Client
	cacheDir    string
	mu          sync.RWMutex
	years       map[int]*yearData

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
func (m *holidayManager) Start(_ context.Context) error {
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
