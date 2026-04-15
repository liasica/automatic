// Copyright (C) automatic. 2026-present.
//
// Created at 2026-04-15, by liasica

package feishu

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// 打卡类型常量，用于 NotifyPunchSuccess 区分上下班
const (
	PunchTypeCheckIn  = "checkin"
	PunchTypeCheckOut = "checkout"
)

// 打卡来源标签，用于推送给用户的提示文案
const (
	PunchSourceAuto       = "自动打卡"
	PunchSourceManual     = "手动添加"
	PunchSourceManualCLI  = "手动添加(命令行)"
	PunchSourceRetry      = "失败重试"
)

// SendText 向指定用户发送文本消息，receiveId 使用 user_id
func (feishu *Feishu) SendText(userId string, content string) error {
	payload, err := sonic.MarshalString(map[string]string{"text": content})
	if err != nil {
		return fmt.Errorf("序列化文本消息失败: %w", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeUserId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(userId).
			MsgType("text").
			Content(payload).
			Build()).
		Build()

	_, err = parseResponse(feishu.client.Im.V1.Message.Create(context.Background(), req))
	return err
}

// NotifyPunchSuccess 打卡成功后向用户发送飞书通知
// 失败时仅打印错误日志，不影响主流程
func (feishu *Feishu) NotifyPunchSuccess(userId string, punchType string, checkTime time.Time, source string) error {
	// 上班用日出、下班用日落，方便一眼区分
	icon := "🌅"
	label := "上班打卡"
	if punchType == PunchTypeCheckOut {
		icon = "🌆"
		label = "下班打卡"
	}
	content := fmt.Sprintf("%s %s成功\n时间：%s\n来源：%s", icon, label, checkTime.Format(time.DateTime), source)
	return feishu.SendText(userId, content)
}
