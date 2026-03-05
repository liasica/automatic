// Copyright (C) automatic. 2026-present.
//
// Created at 2026-02-25, by liasica

package util

import (
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/dromara/carbon/v2"
)

func ParseHourMinute(timeStr string) (hour, minute int, err error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return 0, 0, errors.New("invalid time format, expected 'HH:MM'")
	}

	hour, err = strconv.Atoi(parts[0])
	if err != nil {
		return
	}

	minute, err = strconv.Atoi(parts[1])
	return
}

// TimeStringToMinutes 将时间字符串转换为分钟数
// timeStr 格式为 "HH:MM"，如 "6:30" 表示 390 分钟
func TimeStringToMinutes(timeStr string) (int, error) {
	parts := len(timeStr)
	// 简单的验证
	if parts < 4 {
		return 0, errors.New("invalid time format")
	}

	// 手动解析时间字符串
	var hours, minutes int
	_, err := time.Parse("15:04", timeStr)
	if err != nil {
		return 0, err
	}

	// 重新解析得到小时和分钟
	t, _ := time.Parse("15:04", timeStr)
	hours = t.Hour()
	minutes = t.Minute()

	return hours*60 + minutes, nil
}

// ParseTimeRange 解析时间范围字符串
// timeRangeStr 格式为 "HH:MM-HH:MM"，如 "6:30-7:30"
func ParseTimeRange(timeRangeStr string) (start, end int, err error) {
	parts := strings.Split(timeRangeStr, "-")
	if len(parts) != 2 {
		return 0, 0, errors.New("invalid time range format, expected 'HH:MM-HH:MM'")
	}

	start, err = TimeStringToMinutes(parts[0])
	if err != nil {
		return 0, 0, err
	}

	end, err = TimeStringToMinutes(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return start, end, nil
}

// GenerateFlowTimes 根据给定的日期字符串和自定义时间范围随机生成上班和下班时间
// dateStr 日期字符串格式：2006-01-02
// checkInRangeStr 上班时间范围字符串，格式 "HH:MM-HH:MM"，如 "6:30-7:30"，为空时使用默认值 "6:30-7:30"
// checkOutRangeStr 下班时间范围字符串，格式 "HH:MM-HH:MM"，如 "20:00-22:30"，为空时使用默认值 "20:00-22:30"
// 秒钟随机生成，范围 1-59（禁止0秒）
func GenerateFlowTimes(dateStr string, checkInRangeStr string, checkOutRangeStr string) (checkInTime time.Time, checkOutTime time.Time, err error) {
	// 使用 carbon 解析日期字符串
	date := carbon.Parse(dateStr, time.Local.String())
	if date.Error != nil {
		return time.Time{}, time.Time{}, date.Error
	}

	// 将 carbon 时间转换为标准 time.Time
	baseTime := date.StdTime()

	// 使用默认值（如果未提供）
	if checkInRangeStr == "" {
		checkInRangeStr = "6:30-7:30"
	}
	if checkOutRangeStr == "" {
		checkOutRangeStr = "20:00-22:30"
	}

	// 解析上班时间范围
	var checkInStartMinutes, checkInEndMinutes int
	checkInStartMinutes, checkInEndMinutes, err = ParseTimeRange(checkInRangeStr)
	if err != nil {
		return
	}

	// 解析下班时间范围
	var checkOutStartMinutes, checkOutEndMinutes int
	checkOutStartMinutes, checkOutEndMinutes, err = ParseTimeRange(checkOutRangeStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	// 随机生成上班时间
	checkInRangeMinutes := checkInEndMinutes - checkInStartMinutes
	checkInMinutes := checkInStartMinutes + rand.Intn(checkInRangeMinutes+1)
	checkInSeconds := 1 + rand.Intn(59) // 随机秒数：1-59（禁止0秒）
	checkInDuration := time.Duration(checkInMinutes)*time.Minute + time.Duration(checkInSeconds)*time.Second
	checkInTime = baseTime.Add(checkInDuration)

	// 随机生成下班时间
	checkOutRangeMinutes := checkOutEndMinutes - checkOutStartMinutes
	checkOutMinutes := checkOutStartMinutes + rand.Intn(checkOutRangeMinutes+1)
	checkOutSeconds := 1 + rand.Intn(59) // 随机秒数：1-59（禁止0秒）
	checkOutDuration := time.Duration(checkOutMinutes)*time.Minute + time.Duration(checkOutSeconds)*time.Second
	checkOutTime = baseTime.Add(checkOutDuration)

	return checkInTime, checkOutTime, nil
}
