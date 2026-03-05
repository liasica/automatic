// Copyright (C) automatic. 2026-present.
//
// Created at 2026-02-25, by liasica

package command

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dromara/carbon/v2"
	larkattendance "github.com/larksuite/oapi-sdk-go/v3/service/attendance/v1"
	"github.com/urfave/cli/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"automatic/core"
	"automatic/di"
	"automatic/integration/feishu"
	"automatic/integration/openwrt"
)

type Punch struct {
	*cli.Command
}

func NewPunch() (p *Punch) {
	p = &Punch{}
	p.Command = &cli.Command{
		Name:        "punch",
		Usage:       "打卡工具",
		Description: "可以实现自动打卡、手动添加打卡等功能",
		Commands: []*cli.Command{
			p.Query(),
			p.Add(),
			p.Del(),
			p.Run(),
		},
	}
	return
}

// Query 查询打卡记录
func (p *Punch) Query() *cli.Command {
	return &cli.Command{
		Name:  "query",
		Usage: "查询打卡记录",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "user",
				Aliases:  []string{"u"},
				Usage:    "飞书用户ID，格式为飞书用户 userId",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "from",
				Aliases:  []string{"f"},
				Usage:    "查询日期，格式为 YYYY-MM-DD / YYYY-MM-DD HH:MM:SS",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "to",
				Aliases: []string{"t"},
				Usage:   "查询日期，格式为 YYYY-MM-DD / YYYY-MM-DD HH:MM:SS，默认为查询开始日期当天的23:59:59",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			userId := cmd.String("user")
			fromCarbon := carbon.Parse(cmd.String("from"), time.Local.String())
			if !fromCarbon.IsValid() {
				return fmt.Errorf("无效的开始日期格式，请使用 YYYY-MM-DD 或 YYYY-MM-DD HH:MM:SS")
			}

			toInput := cmd.String("to")
			var toCarbon *carbon.Carbon
			if toInput == "" {
				toCarbon = fromCarbon.EndOfDay()
			} else {
				toCarbon = carbon.Parse(toInput, time.Local.String())
				if !toCarbon.IsValid() {
					return fmt.Errorf("无效的结束日期格式，请使用 YYYY-MM-DD 或 YYYY-MM-DD HH:MM:SS")
				}
			}

			zap.S().Infof("查询打卡记录，用户：%s，日期：%s", userId, fmt.Sprintf("%s ~ %s", fromCarbon, toCarbon))

			return di.New(cmd, fx.Invoke(func(fs *feishu.Feishu) error {
				result, err := fs.UserFlowsQuery(userId, fromCarbon.StdTime(), toCarbon.StdTime())
				if err != nil {
					return err
				}

				for _, record := range result.UserFlowResults {
					i, err := strconv.ParseInt(*record.CheckTime, 10, 64)
					if err != nil {
						return err
					}

					zap.S().Infof("打卡记录 - ID: %s, 时间: %s", *record.RecordId, time.Unix(i, 0).Format("2006-01-02 15:04:05"))
				}

				return nil
			})).Start(ctx)
		},
	}
}

// Add 添加手动打卡记录
func (p *Punch) Add() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "手动添加打卡记录",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "time",
				Aliases:  []string{"t"},
				Usage:    "打卡时间，格式为 YYYY-MM-DD HH:MM:SS",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "user",
				Aliases:  []string{"u"},
				Usage:    "飞书用户ID，格式为飞书用户 userId",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			t := carbon.Parse(cmd.String("time"), time.Local.String())
			if !t.IsValid() {
				return fmt.Errorf("无效的时间格式，请使用 YYYY-MM-DD HH:MM:SS")
			}
			userId := cmd.String("user")

			zap.S().Infof("手动添加打卡记录，用户：%s，时间：%s", userId, t)

			return di.New(cmd, fx.Invoke(func(fs *feishu.Feishu) error {
				return fs.UserFlowsCreate(userId, t.StdTime())
			})).Start(ctx)
		},
	}
}

// Del 删除打卡记录
func (p *Punch) Del() *cli.Command {
	return &cli.Command{
		Name:  "del",
		Usage: "删除打卡记录",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:     "ids",
				Usage:    "打卡记录ID，格式为飞书打卡记录 [recordId]",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			ids := cmd.StringSlice("ids")

			zap.S().Infof("删除打卡记录，记录ID：%s", ids)

			return di.New(cmd, fx.Invoke(func(fs *feishu.Feishu) error {
				successRecordIDs, failedRecordIDs, err := fs.UserFlowsDelete(ids...)
				if err != nil {
					return err
				}

				for _, id := range successRecordIDs {
					zap.S().Infof("成功删除打卡记录，记录ID：%s", id)
				}

				for _, id := range failedRecordIDs {
					zap.S().Errorf("删除打卡记录失败，记录ID：%s", id)
				}

				return nil
			})).Start(ctx)
		},
	}
}

// Run 启动自动打卡
func (p *Punch) Run() *cli.Command {
	return &cli.Command{
		Name:        "run",
		Usage:       "开始自动打卡",
		Description: "根据配置文件中的设置，开始自动打卡",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			app := di.New(cmd, fx.Invoke(func(lc fx.Lifecycle, cfg *core.Config, cache *core.Cache, fs *feishu.Feishu, ow *openwrt.OpenWrt) {
				lc.Append(fx.Hook{
					OnStart: func(context.Context) error {
						ow.AddHandler(func(event openwrt.DeviceEvent) {
							p.doEvent(ctx, cfg, cache, fs, event)
						})
						ow.Start(1 * time.Minute)
						zap.S().Infof("自动打卡服务已启动，轮询间隔: %s", time.Minute)
						return nil
					},
					OnStop: func(context.Context) error {
						ow.Stop()
						zap.S().Info("自动打卡服务已停止")
						return nil
					},
				})
			}))
			if err := app.Start(ctx); err != nil {
				return err
			}

			ch := make(chan os.Signal, 1)
			signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(ch)

			select {
			case <-ctx.Done():
			case sig := <-ch:
				zap.S().Infof("收到退出信号: %s", sig.String())
			}

			stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return app.Stop(stopCtx)
		},
	}
}

func (p *Punch) doEvent(ctx context.Context, cfg *core.Config, cache *core.Cache, fs *feishu.Feishu, event openwrt.DeviceEvent) {
	if event.Device == nil {
		return
	}

	user := p.matchUser(cfg.Users, event.Device.Mac)
	if user == nil {
		return
	}

	now := time.Now()
	// 在线事件触发上班打卡，离线事件触发下班打卡。
	switch event.Type {
	case openwrt.EventOnline:
		p.tryCheckIn(ctx, cache, fs, event, user, now)
	case openwrt.EventOffline:
		p.tryCheckOut(ctx, cache, fs, event, user, now)
	default:
		return
	}
}

func (p *Punch) matchUser(users []*core.User, mac string) *core.User {
	i := slices.IndexFunc(users, func(u *core.User) bool {
		return slices.ContainsFunc(u.MacAddresses, func(m string) bool {
			return strings.EqualFold(m, mac)
		})
	})
	if i == -1 {
		return nil
	}
	return users[i]
}

func (p *Punch) tryCheckIn(ctx context.Context, cache *core.Cache, fs *feishu.Feishu, event openwrt.DeviceEvent, user *core.User, now time.Time) {
	latestCheckInTime := p.dayAtHourMinute(now, user.CheckIn.LatestTime.Hour, user.CheckIn.LatestTime.Minute)
	// 仅当设备上线时间不晚于配置的 latest，才进入上班打卡后续判定。
	if now.After(latestCheckInTime) {
		return
	}

	key := p.punchCacheKey("checkin", user.Id, now)
	// Redis key 用于“用户+日期+打卡类型”幂等，避免同一天重复触发。
	exists, err := cache.Exists(ctx, key)
	if err != nil {
		zap.S().Errorf("读取上班打卡缓存失败，key: %s, error: %v", key, err)
		return
	}
	if exists {
		return
	}

	var data *larkattendance.QueryUserFlowRespData
	// 上班打卡去重：仅看当日 00:00:00 到 09:00:00 是否已有记录。
	data, err = fs.UserFlowsQuery(user.Id, p.dayStart(now), p.dayAtHour(now, 9))
	if err != nil {
		zap.S().Errorf("查询上班打卡记录失败，user: %s, error: %v", user.Id, err)
		return
	}

	// 上班打卡时间 = 设备上线时间向前随机 [from, to] 分钟。
	checkTime := p.randomCheckInTime(user, now)

	// 已有上班记录则仅写入幂等标记，后续事件直接跳过。
	if p.hasCheckInRecord(data) {
		err = cache.Set(ctx, key, checkTime.Format(time.DateTime), 24*time.Hour)
		if err != nil {
			zap.S().Errorf("写入上班打卡缓存失败，key: %s, error: %v", key, err)
		}
		return
	}

	zap.S().Infof("自动上班打卡，user: %s, mac: %s, time: %s", user.Id, event.Device.Mac, checkTime.Format(time.DateTime))

	err = fs.UserFlowsCreate(user.Id, checkTime)
	if err != nil {
		zap.S().Errorf("自动上班打卡失败，user: %s, mac: %s, time: %s, error: %v", user.Id, event.Device.Mac, checkTime.Format(time.DateTime), err)
		return
	}

	err = cache.Set(ctx, key, strconv.FormatInt(checkTime.Unix(), 10), 24*time.Hour)
	if err != nil {
		zap.S().Errorf("写入上班打卡缓存失败，key: %s, error: %v", key, err)
	}
	zap.S().Infof("自动上班打卡成功，user: %s, mac: %s, time: %s", user.Id, event.Device.Mac, checkTime.Format(time.DateTime))
}

func (p *Punch) tryCheckOut(ctx context.Context, cache *core.Cache, fs *feishu.Feishu, event openwrt.DeviceEvent, user *core.User, now time.Time) {
	earliestCheckOutTime := p.dayAtHourMinute(now, user.CheckOut.EarliestTime.Hour, user.CheckOut.EarliestTime.Minute)
	// 仅当设备离线时间不早于配置的 earliest，才进入下班打卡后续判定。
	if now.Before(earliestCheckOutTime) {
		return
	}

	key := p.punchCacheKey("checkout", user.Id, now)
	// Redis key 用于“用户+日期+打卡类型”幂等，避免同一天重复触发。
	exists, err := cache.Exists(ctx, key)
	if err != nil {
		zap.S().Errorf("读取下班打卡缓存失败，key: %s, error: %v", key, err)
		return
	}
	if exists {
		return
	}

	var data *larkattendance.QueryUserFlowRespData
	// 下班打卡去重：仅看当日 18:00:00 到 24:00:00 是否已有记录。
	data, err = fs.UserFlowsQuery(user.Id, p.dayAtHour(now, 18), p.dayEnd(now))
	if err != nil {
		zap.S().Errorf("查询下班打卡记录失败，user: %s, error: %v", user.Id, err)
		return
	}

	// 下班打卡时间 = 设备离线时间向后随机 [from, to] 分钟。
	checkTime := p.randomCheckOutTime(user, now)

	// 已有下班记录则仅写入幂等标记，后续事件直接跳过。
	if p.hasCheckOutRecord(data) {
		err = cache.Set(ctx, key, checkTime.Format(time.DateTime), 24*time.Hour)
		if err != nil {
			zap.S().Errorf("写入下班打卡缓存失败，key: %s, error: %v", key, err)
		}
		return
	}

	zap.S().Infof("准备自动下班打卡，user: %s, mac: %s, time: %s", user.Id, event.Device.Mac, checkTime.Format(time.DateTime))

	err = fs.UserFlowsCreate(user.Id, checkTime)
	if err != nil {
		zap.S().Errorf("自动下班打卡失败，user: %s, mac: %s, time: %s, error: %v", user.Id, event.Device.Mac, checkTime.Format(time.DateTime), err)
		return
	}

	err = cache.Set(ctx, key, strconv.FormatInt(checkTime.Unix(), 10), 24*time.Hour)
	if err != nil {
		zap.S().Errorf("写入下班打卡缓存失败，key: %s, error: %v", key, err)
	}
	zap.S().Infof("自动下班打卡成功，user: %s, mac: %s, time: %s", user.Id, event.Device.Mac, checkTime.Format(time.DateTime))
}

func (p *Punch) hasCheckInRecord(data *larkattendance.QueryUserFlowRespData) bool {
	if data == nil {
		return false
	}

	for _, flow := range data.UserFlowResults {
		if flow == nil || flow.CheckTime == nil {
			continue
		}
		return true
	}
	return false
}

func (p *Punch) hasCheckOutRecord(data *larkattendance.QueryUserFlowRespData) bool {
	if data == nil {
		return false
	}

	for _, flow := range data.UserFlowResults {
		if flow == nil || flow.CheckTime == nil {
			continue
		}
		return true
	}
	return false
}

func (p *Punch) randomCheckInTime(user *core.User, baseTime time.Time) time.Time {
	offset := randomInRange(user.CheckIn.From, user.CheckIn.To)
	return baseTime.Add(-time.Duration(offset) * time.Minute)
}

func (p *Punch) randomCheckOutTime(user *core.User, baseTime time.Time) time.Time {
	offset := randomInRange(user.CheckOut.From, user.CheckOut.To)
	return baseTime.Add(time.Duration(offset) * time.Minute)
}

func (p *Punch) dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func (p *Punch) dayAtHour(t time.Time, hour int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), hour, 0, 0, 0, t.Location())
}

func (p *Punch) dayEnd(t time.Time) time.Time {
	return p.dayStart(t).Add(24 * time.Hour)
}

func (p *Punch) dayAtHourMinute(t time.Time, hour int, minute int) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), hour, minute, 0, 0, t.Location())
}

func (p *Punch) punchCacheKey(kind, userId string, now time.Time) string {
	return fmt.Sprintf("automatic:punch:%s:%s:%s", kind, now.Format("20060102"), userId)
}

func normalizeOffsets(firstOffset, secondOffset int) (int, int) {
	if firstOffset > secondOffset {
		return secondOffset, firstOffset
	}
	return firstOffset, secondOffset
}

func randomInRange(firstOffset, secondOffset int) int {
	lowerOffset, upperOffset := normalizeOffsets(firstOffset, secondOffset)
	if lowerOffset == upperOffset {
		return lowerOffset
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return lowerOffset + r.Intn(upperOffset-lowerOffset+1)
}
