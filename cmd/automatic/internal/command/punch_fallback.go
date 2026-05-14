// Copyright (C) automatic. 2026-present.
//
// Created at 2026-05-14, by liasica

package command

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
	"time"

	larkattendance "github.com/larksuite/oapi-sdk-go/v3/service/attendance/v1"
	"go.uber.org/zap"

	"automatic/core"
	"automatic/integration/feishu"
)

// 兜底打卡触发时间（每天的本地时间）
// 9 点之后还未上班打卡 → 触发上班兜底；21 点之后有上班无下班 → 触发下班兜底
const (
	fallbackCheckInTriggerHour  = 9
	fallbackCheckOutTriggerHour = 21
)

// 兜底打卡时间窗口（每天的本地时间，[from, to] 闭区间）
// 上班窗口落在 07:00 - 07:30；下班窗口落在 19:50 - 20:40
var (
	fallbackCheckInWindowFrom  = clockTime{Hour: 7, Minute: 0, Second: 0}
	fallbackCheckInWindowTo    = clockTime{Hour: 7, Minute: 30, Second: 0}
	fallbackCheckOutWindowFrom = clockTime{Hour: 19, Minute: 50, Second: 0}
	fallbackCheckOutWindowTo   = clockTime{Hour: 20, Minute: 40, Second: 0}
)

// clockTime 描述某一天内的时刻（不含日期）
type clockTime struct {
	Hour   int
	Minute int
	Second int
}

// at 将 clockTime 落到指定日期上
func (c clockTime) at(day time.Time) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), c.Hour, c.Minute, c.Second, 0, day.Location())
}

// startFallbackScheduler 启动兜底打卡调度协程
// 在 OnStart 中调用，OnStop 时通过 stopCh 关闭
func (p *Punch) startFallbackScheduler(cfg *core.Config, cache *core.Cache, fs *feishu.Feishu, stopCh <-chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.fallbackLoop(cfg, cache, fs, stopCh)
	}()
}

// fallbackLoop 调度循环：每次睡到下一个最近的触发点（09:00 或 21:00），唤醒后执行对应兜底
func (p *Punch) fallbackLoop(cfg *core.Config, cache *core.Cache, fs *feishu.Feishu, stopCh <-chan struct{}) {
	for {
		now := time.Now()
		nextCheckIn := nextDailyTime(now, fallbackCheckInTriggerHour, 0, 0)
		nextCheckOut := nextDailyTime(now, fallbackCheckOutTriggerHour, 0, 0)

		var (
			fireAt time.Time
			isIn   bool
		)
		if nextCheckIn.Before(nextCheckOut) {
			fireAt = nextCheckIn
			isIn = true
		} else {
			fireAt = nextCheckOut
			isIn = false
		}

		timer := time.NewTimer(time.Until(fireAt))
		select {
		case <-stopCh:
			timer.Stop()
			return
		case <-timer.C:
			fireTime := time.Now()
			// 仅工作日执行兜底
			if core.GetDayType(fireTime) != core.DayTypeWorkday {
				zap.S().Infof("兜底打卡跳过，非工作日，time: %s, dayType: %s", fireTime.Format(time.DateTime), core.GetDayType(fireTime))
				continue
			}

			if isIn {
				p.runCheckInFallback(cfg, cache, fs, fireTime)
			} else {
				p.runCheckOutFallback(cfg, cache, fs, fireTime)
			}
		}
	}
}

// runCheckInFallback 遍历所有用户执行上班兜底
func (p *Punch) runCheckInFallback(cfg *core.Config, cache *core.Cache, fs *feishu.Feishu, now time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	for _, user := range cfg.Users {
		p.tryFallbackCheckIn(ctx, cache, fs, user, now)
	}
}

// runCheckOutFallback 遍历所有用户执行下班兜底
func (p *Punch) runCheckOutFallback(cfg *core.Config, cache *core.Cache, fs *feishu.Feishu, now time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	for _, user := range cfg.Users {
		p.tryFallbackCheckOut(ctx, cache, fs, user, now)
	}
}

// tryFallbackCheckIn 上班兜底：当日 00:00 - 09:00 无上班记录则随机在 07:00 - 07:30 打卡
// 复用 tryCheckIn 的幂等键，避免与设备事件触发的上班打卡互相冲突
func (p *Punch) tryFallbackCheckIn(ctx context.Context, cache *core.Cache, fs *feishu.Feishu, user *core.User, now time.Time) {
	key := p.punchCacheKey("checkin", user.Id, now)
	// 幂等键已存在说明今天已有正常或兜底流程介入，直接跳过
	acquired, err := cache.SetIfAbsent(ctx, key, "pending", 24*time.Hour)
	if err != nil {
		zap.S().Errorf("抢占上班兜底打卡幂等键失败，key: %s, error: %v", key, err)
		return
	}
	if !acquired {
		return
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		if delErr := cache.Del(ctx, key); delErr != nil {
			zap.S().Errorf("回滚上班兜底打卡幂等键失败，key: %s, error: %v", key, delErr)
		}
	}()

	// 与常规上班打卡保持一致：只看当日 00:00 - 09:00 是否已有记录
	var data *larkattendance.QueryUserFlowRespData
	data, err = fs.UserFlowsQuery(user.Id, p.dayStart(now), p.dayAtHour(now, fallbackCheckInTriggerHour))
	if err != nil {
		zap.S().Errorf("查询上班打卡记录失败（兜底），user: %s, error: %v", user.Id, err)
		return
	}
	if p.hasCheckInRecord(data) {
		// 已有正常上班打卡记录，仅写入幂等标记
		committed = true
		if setErr := cache.Set(ctx, key, "skip:has_checkin_record", 24*time.Hour); setErr != nil {
			zap.S().Errorf("写入上班兜底打卡缓存失败，key: %s, error: %v", key, setErr)
		}
		return
	}

	checkTime := randomTimeInWindow(now, fallbackCheckInWindowFrom, fallbackCheckInWindowTo)
	zap.S().Infof("自动上班兜底打卡，user: %s, time: %s", user.Id, checkTime.Format(time.DateTime))

	err = fs.UserFlowsCreate(user.Id, checkTime)
	if err != nil {
		failedReq := p.newFailedPunchRequest("checkin", user.Id, "", checkTime, "fallback_checkin", err)
		recordErr := p.recordFailedPunch(ctx, cache, failedReq)
		if recordErr != nil {
			zap.S().Errorf("自动上班兜底打卡失败，且记录失败请求失败，user: %s, time: %s, error: %v, recordError: %v", user.Id, checkTime.Format(time.DateTime), err, recordErr)
			return
		}
		committed = true
		zap.S().Warnf("自动上班兜底打卡失败，已记录失败请求，requestId: %s, user: %s, time: %s, error: %v", failedReq.RequestID, user.Id, checkTime.Format(time.DateTime), err)
		return
	}

	committed = true
	if setErr := cache.Set(ctx, key, strconv.FormatInt(checkTime.Unix(), 10), 24*time.Hour); setErr != nil {
		zap.S().Errorf("写入上班兜底打卡缓存失败，key: %s, error: %v", key, setErr)
	}
	zap.S().Infof("自动上班兜底打卡成功，user: %s, time: %s", user.Id, checkTime.Format(time.DateTime))
	if notifyErr := fs.NotifyPunchSuccess(user.Id, feishu.PunchTypeCheckIn, checkTime, feishu.PunchSourceAuto); notifyErr != nil {
		zap.S().Warnf("自动上班兜底打卡成功通知发送失败，user: %s, error: %v", user.Id, notifyErr)
	}
}

// tryFallbackCheckOut 下班兜底：当日有上班记录且 18:00 - 24:00 无下班记录则随机在 19:50 - 20:40 打卡
// 复用 tryCheckOut 的幂等键，避免与设备事件触发的下班打卡互相冲突
func (p *Punch) tryFallbackCheckOut(ctx context.Context, cache *core.Cache, fs *feishu.Feishu, user *core.User, now time.Time) {
	key := p.punchCacheKey("checkout", user.Id, now)
	acquired, err := cache.SetIfAbsent(ctx, key, "pending", 24*time.Hour)
	if err != nil {
		zap.S().Errorf("抢占下班兜底打卡幂等键失败，key: %s, error: %v", key, err)
		return
	}
	if !acquired {
		return
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		if delErr := cache.Del(ctx, key); delErr != nil {
			zap.S().Errorf("回滚下班兜底打卡幂等键失败，key: %s, error: %v", key, delErr)
		}
	}()

	// 与常规下班打卡保持一致：先查 00:00 - earliestCheckOutTime 是否有上班记录
	earliestCheckOutTime := p.dayAtHourMinute(now, user.CheckOut.EarliestTime.Hour, user.CheckOut.EarliestTime.Minute)
	var checkInData *larkattendance.QueryUserFlowRespData
	checkInData, err = fs.UserFlowsQuery(user.Id, p.dayStart(now), earliestCheckOutTime)
	if err != nil {
		zap.S().Errorf("查询上班打卡记录失败（下班兜底），user: %s, error: %v", user.Id, err)
		return
	}
	if !p.hasCheckInRecord(checkInData) {
		committed = true
		if setErr := cache.Set(ctx, key, "skip:no_checkin_record", 24*time.Hour); setErr != nil {
			zap.S().Errorf("写入下班兜底打卡缓存失败，key: %s, error: %v", key, setErr)
		}
		zap.S().Infof("自动下班兜底打卡跳过，未发现上班打卡记录，user: %s", user.Id)
		return
	}

	// 下班打卡去重：仅看当日 18:00 - 24:00 是否已有记录
	var checkOutData *larkattendance.QueryUserFlowRespData
	checkOutData, err = fs.UserFlowsQuery(user.Id, p.dayAtHour(now, 18), p.dayEnd(now))
	if err != nil {
		zap.S().Errorf("查询下班打卡记录失败（兜底），user: %s, error: %v", user.Id, err)
		return
	}
	if p.hasCheckOutRecord(checkOutData) {
		committed = true
		if setErr := cache.Set(ctx, key, "skip:has_checkout_record", 24*time.Hour); setErr != nil {
			zap.S().Errorf("写入下班兜底打卡缓存失败，key: %s, error: %v", key, setErr)
		}
		return
	}

	checkTime := randomTimeInWindow(now, fallbackCheckOutWindowFrom, fallbackCheckOutWindowTo)
	zap.S().Infof("自动下班兜底打卡，user: %s, time: %s", user.Id, checkTime.Format(time.DateTime))

	err = fs.UserFlowsCreate(user.Id, checkTime)
	if err != nil {
		failedReq := p.newFailedPunchRequest("checkout", user.Id, "", checkTime, "fallback_checkout", err)
		recordErr := p.recordFailedPunch(ctx, cache, failedReq)
		if recordErr != nil {
			zap.S().Errorf("自动下班兜底打卡失败，且记录失败请求失败，user: %s, time: %s, error: %v, recordError: %v", user.Id, checkTime.Format(time.DateTime), err, recordErr)
			return
		}
		committed = true
		zap.S().Warnf("自动下班兜底打卡失败，已记录失败请求，requestId: %s, user: %s, time: %s, error: %v", failedReq.RequestID, user.Id, checkTime.Format(time.DateTime), err)
		return
	}

	committed = true
	if setErr := cache.Set(ctx, key, strconv.FormatInt(checkTime.Unix(), 10), 24*time.Hour); setErr != nil {
		zap.S().Errorf("写入下班兜底打卡缓存失败，key: %s, error: %v", key, setErr)
	}
	zap.S().Infof("自动下班兜底打卡成功，user: %s, time: %s", user.Id, checkTime.Format(time.DateTime))
	if notifyErr := fs.NotifyPunchSuccess(user.Id, feishu.PunchTypeCheckOut, checkTime, feishu.PunchSourceAuto); notifyErr != nil {
		zap.S().Warnf("自动下班兜底打卡成功通知发送失败，user: %s, error: %v", user.Id, notifyErr)
	}
}

// nextDailyTime 返回 now 之后最近一次 hh:mm:ss 的本地时间
func nextDailyTime(now time.Time, hour, minute, second int) time.Time {
	loc := now.Location()
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, second, 0, loc)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// randomTimeInWindow 在 [from, to] 闭区间内（按当日时间）随机取一个时间点
func randomTimeInWindow(day time.Time, from, to clockTime) time.Time {
	fromAt := from.at(day)
	toAt := to.at(day)
	if !toAt.After(fromAt) {
		return fromAt
	}
	spanSec := int64(toAt.Sub(fromAt).Seconds())
	return fromAt.Add(time.Duration(rand.Int63n(spanSec+1)) * time.Second)
}
