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
