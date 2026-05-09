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
