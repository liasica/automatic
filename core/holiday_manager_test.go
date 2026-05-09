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
