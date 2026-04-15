// Copyright (C) automatic. 2026-present.
//
// Created at 2026-04-15, by liasica

package feishu

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestSendText 向 config.yaml 中的第一个用户发送一条测试文本消息
func TestSendText(t *testing.T) {
	l, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(l)

	cfg, cache := getConfig(t)
	require.NotEmpty(t, cfg.Users, "config.yaml 中至少需要配置一个用户")

	fs := New(cfg, cache)
	userId := cfg.Users[0].Id
	err := fs.SendText(userId, "打卡通知测试：普通文本消息")
	require.NoError(t, err)
}

// TestNotifyPunchSuccess 验证上/下班打卡通知的文案渲染与送达
func TestNotifyPunchSuccess(t *testing.T) {
	l, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(l)

	cfg, cache := getConfig(t)
	require.NotEmpty(t, cfg.Users, "config.yaml 中至少需要配置一个用户")

	fs := New(cfg, cache)
	userId := cfg.Users[0].Id
	now := time.Now()

	err := fs.NotifyPunchSuccess(userId, PunchTypeCheckIn, now, PunchSourceAuto)
	require.NoError(t, err)

	err = fs.NotifyPunchSuccess(userId, PunchTypeCheckOut, now.Add(10*time.Hour), PunchSourceManual)
	require.NoError(t, err)
}
