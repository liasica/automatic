// Copyright (C) automatic. 2026-present.
//
// Created at 2026-02-22, by liasica

package feishu

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"testing"
	"time"

	larkattendance "github.com/larksuite/oapi-sdk-go/v3/service/attendance/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"automatic/core"
	"automatic/util"
)

func getConfig(t *testing.T) (*core.Config, *core.Cache) {
	cfg, err := core.NewConfig("../../configs/config.yaml")
	require.NoError(t, err)

	var cache *core.Cache
	cache, err = core.NewCache(cfg.Redis.Addr, cfg.Redis.DB)
	require.NoError(t, err)

	return cfg, cache
}

var dateStr = ""

func TestUserFlowsCreate(t *testing.T) { // 设置全局时区
	tz := "Asia/Shanghai"
	_ = os.Setenv("TZ", tz)
	loc, _ := time.LoadLocation(tz)
	time.Local = loc

	l, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(l)

	if dateStr == "" {
		dateStr = time.Now().Format("20060102")
	}

	t1, t2, err := util.GenerateFlowTimes(dateStr, "5:30-5:45", "19:59-20:35")
	require.NoError(t, err)
	t.Log(t1, t2)

	lar := New(getConfig(t))
	// err = lar.UserFlowsCreate("liasica", t1)
	// require.NoError(t, err)
	//
	err = lar.UserFlowsCreate("liasica", t2)
	require.NoError(t, err)
}

func TestUserFlowsQuery(t *testing.T) {
	tz := "Asia/Shanghai"
	_ = os.Setenv("TZ", tz)
	loc, _ := time.LoadLocation(tz)
	time.Local = loc

	l, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(l)

	lar := New(getConfig(t))
	resp, err := lar.UserFlowsQuery("liasica", time.Date(2026, 2, 27, 0, 0, 0, 0, time.Local), time.Date(2026, 2, 28, 0, 0, 0, 0, time.Local))
	require.NoError(t, err)
	t.Logf("Flows: %d", len(resp.UserFlowResults))
	for _, result := range resp.UserFlowResults {
		t.Logf("RecordId: %s, CheckIn: %s, LocationName: %s, Comment: %s", *result.RecordId, *result.CheckTime, *result.LocationName, *result.Comment)
	}
}

func TestUserFlowsDelete(t *testing.T) {
	tz := "Asia/Shanghai"
	_ = os.Setenv("TZ", tz)
	loc, _ := time.LoadLocation(tz)
	time.Local = loc

	l, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(l)

	lar := New(getConfig(t))
	flows, err := lar.UserFlowsQuery("liasica", time.Date(2026, 2, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 2, 25, 23, 59, 0, 0, time.Local))
	require.NoError(t, err)
	t.Logf("Flows: %d", len(flows.UserFlowResults))

	// 查找当日重复的
	type repeated struct {
		CheckTime time.Time
		Flow      *larkattendance.UserFlow
	}
	m := make(map[string][]*repeated)
	for _, f := range flows.UserFlowResults {
		if f.CheckTime == nil || f.RecordId == nil {
			continue
		}

		var ct int64
		ct, err = strconv.ParseInt(*f.CheckTime, 10, 64)
		require.NoError(t, err)

		checkTime := time.Unix(ct, 0)
		d := checkTime.Format("2006-01-02 PM")
		m[d] = append(m[d], &repeated{
			CheckTime: checkTime,
			Flow:      f,
		})
	}

	var needDelete []string
	for d, ss := range m {
		if len(ss) > 1 {
			// 排序
			slices.SortFunc(ss, func(a, b *repeated) int {
				if a.CheckTime.Before(b.CheckTime) {
					return -1
				}
				if a.CheckTime.After(b.CheckTime) {
					return 1
				}
				return 0
			})

			for i := 1; i < len(ss); i++ {
				fmt.Printf("需要删除 [%s]，打卡时间： %s\t%s\n", *ss[i].Flow.RecordId, d, ss[i].CheckTime)
				needDelete = append(needDelete, *ss[i].Flow.RecordId)
			}
		}
	}

	// 调用删除
	if len(needDelete) > 0 {
		var success, fail []string
		success, fail, err = lar.UserFlowsDelete(needDelete...)
		require.NoError(t, err)
		t.Logf("成功删除 %d 条，失败 %d 条", len(success), len(fail))
	}
}
