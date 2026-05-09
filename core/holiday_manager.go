// Copyright (C) automatic. 2026-present.
//
// Created at 2026-05-09, by liasica

package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
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
