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
