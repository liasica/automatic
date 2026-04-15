// Copyright (C) automatic. 2026-present.
//
// Created at 2026-04-15, by liasica

package core

import "time"

const (
	DayTypeWorkday = "workday" // 工作日
	DayTypeWeekend = "weekend" // 周末
	DayTypeHoliday = "holiday" // 法定节假日
)

// 国家法定节假日（每年根据国务院发布的放假通知更新）
var nationalHolidays = map[string]bool{
	// ── 2025 ──
	"2025-01-01": true, // 元旦
	"2025-01-28": true, // 春节
	"2025-01-29": true,
	"2025-01-30": true,
	"2025-01-31": true,
	"2025-02-01": true,
	"2025-02-02": true,
	"2025-02-03": true,
	"2025-02-04": true,
	"2025-04-04": true, // 清明节
	"2025-04-05": true,
	"2025-04-06": true,
	"2025-05-01": true, // 劳动节
	"2025-05-02": true,
	"2025-05-03": true,
	"2025-05-04": true,
	"2025-05-05": true,
	"2025-05-31": true, // 端午节
	"2025-06-01": true,
	"2025-06-02": true,
	"2025-10-01": true, // 国庆节+中秋节
	"2025-10-02": true,
	"2025-10-03": true,
	"2025-10-04": true,
	"2025-10-05": true,
	"2025-10-06": true,
	"2025-10-07": true,
	"2025-10-08": true,

	// ── 2026（预估，以国务院正式通知为准）──
	"2026-01-01": true, // 元旦
	"2026-01-02": true,
	"2026-01-03": true,
	"2026-02-15": true, // 春节（正月初一 2/17）
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

// 调休工作日：原本是周末但调整为上班
var makeupWorkdays = map[string]bool{
	// ── 2025 ──
	"2025-01-26": true, // 春节调休
	"2025-02-08": true, // 春节调休
	"2025-04-27": true, // 劳动节调休
	"2025-09-28": true, // 国庆调休
	"2025-10-11": true, // 国庆调休

	// ── 2026（预估）──
	"2026-02-14": true, // 春节调休
	"2026-02-22": true, // 春节调休
	"2026-04-26": true, // 劳动节调休
	"2026-09-27": true, // 国庆调休
	"2026-10-10": true, // 国庆调休
}

// GetDayType 获取日期类型
func GetDayType(t time.Time) string {
	ds := t.Format("2006-01-02")
	if nationalHolidays[ds] {
		return DayTypeHoliday
	}
	if makeupWorkdays[ds] {
		return DayTypeWorkday
	}
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return DayTypeWeekend
	}
	return DayTypeWorkday
}
