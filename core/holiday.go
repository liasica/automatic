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
// manager 已注入 → 走真实节假日数据（来自 holiday-cn 拉取或本地缓存）
// 未注入 → 仅按星期几判定，不识别节假日（裸奔模式，正常生产路径不会走到）
func GetDayType(t time.Time) string {
	if m := getHolidayManager(); m != nil {
		return m.GetDayType(t)
	}
	return weekdayDayType(t)
}

// weekdayDayType 仅依赖 time.Weekday 的判定
// 无节假日数据时使用：周六/周日返回 weekend，其余返回 workday
func weekdayDayType(t time.Time) string {
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return DayTypeWeekend
	}
	return DayTypeWorkday
}
