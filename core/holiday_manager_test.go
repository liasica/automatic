// Copyright (C) automatic. 2026-present.
//
// Created at 2026-05-09, by liasica

package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"resty.dev/v3"
)

// TestWeekdayDayType 仅按星期几判定的纯函数行为
func TestWeekdayDayType(t *testing.T) {
	require.Equal(t, DayTypeWeekend, weekdayDayType(time.Date(2026, 5, 16, 0, 0, 0, 0, time.Local))) // 周六
	require.Equal(t, DayTypeWeekend, weekdayDayType(time.Date(2026, 5, 17, 0, 0, 0, 0, time.Local))) // 周日
	require.Equal(t, DayTypeWorkday, weekdayDayType(time.Date(2026, 5, 11, 0, 0, 0, 0, time.Local))) // 周一
}

// TestGetDayType_NoManager manager 未注入时退化为按星期判定，不识别节假日
// 5/9 是周六，无 manager 时被判为 weekend（接受的代价：A 方案设计选择）
func TestGetDayType_NoManager(t *testing.T) {
	SetHolidayManager(nil) // 清空（防止其他测试残留）
	require.Equal(t, DayTypeWeekend, GetDayType(time.Date(2026, 5, 9, 0, 0, 0, 0, time.Local)))
	require.Equal(t, DayTypeWorkday, GetDayType(time.Date(2026, 5, 11, 0, 0, 0, 0, time.Local)))
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

// TestManagerGetDayType_NoYearData manager 未加载该年时按星期判定，不识别节假日
func TestManagerGetDayType_NoYearData(t *testing.T) {
	m := newTestManager(t.TempDir())
	// 不预置 m.years[2026]，没有数据可用
	require.Equal(t, DayTypeWeekend, m.GetDayType(time.Date(2026, 5, 9, 0, 0, 0, 0, time.Local)))  // 5/9 是周六
	require.Equal(t, DayTypeWorkday, m.GetDayType(time.Date(2026, 5, 11, 0, 0, 0, 0, time.Local))) // 周一
}

// newTestManager 构造一个不启动后台 goroutine 的 manager 用于纯逻辑测试
func newTestManager(cacheDir string) *holidayManager {
	return &holidayManager{
		cacheDir: cacheDir,
		years:    make(map[int]*yearData),
	}
}

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

// TestNextRefreshTime_AfterCutoff 当前时间在 00:05 之后，应返回明日 00:05
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
