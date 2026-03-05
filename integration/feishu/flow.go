// Copyright (C) automatic. 2026-present.
//
// Created at 2026-02-22, by liasica

package feishu

import (
	"context"
	"strconv"
	"time"

	larkattendance "github.com/larksuite/oapi-sdk-go/v3/service/attendance/v1"
)

// UserFlowsCreate 导入用户打卡流水
func (feishu *Feishu) UserFlowsCreate(userId string, checkTime time.Time) error {
	req := larkattendance.NewBatchCreateUserFlowReqBuilder().
		EmployeeType("employee_id").
		Body(larkattendance.NewBatchCreateUserFlowReqBodyBuilder().
			FlowRecords([]*larkattendance.UserFlow{
				larkattendance.NewUserFlowBuilder().
					UserId(userId).
					CreatorId(userId).
					LocationName(`Joash's AX86U_5G(30:5a:3a:c5:51:64)`).
					CheckTime(strconv.FormatInt(checkTime.Unix(), 10)).
					Ssid(`Joash's AX86U_5G`).
					Bssid(`30:5a:3a:c5:51:64`).
					Comment("").
					IsField(false).
					IsWifi(true).
					CheckResult("Normal").
					Build(),
			}).
			Build()).
		Build()

	_, err := parseResponse(feishu.client.Attendance.V1.UserFlow.BatchCreate(context.Background(), req))
	if err != nil {
		return err
	}

	return nil
}

// UserFlowsQuery 查询用户打卡流水
func (feishu *Feishu) UserFlowsQuery(userId string, from, to time.Time) (*larkattendance.QueryUserFlowRespData, error) {
	req := larkattendance.NewQueryUserFlowReqBuilder().
		EmployeeType("employee_id").
		Body(larkattendance.NewQueryUserFlowReqBodyBuilder().
			UserIds([]string{userId}).
			CheckTimeFrom(strconv.FormatInt(from.Unix(), 10)).
			CheckTimeTo(strconv.FormatInt(to.Unix(), 10)).
			Build()).
		Build()

	resp, err := parseResponse(feishu.client.Attendance.V1.UserFlow.Query(context.Background(), req))
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// UserFlowsDelete 删除用户打卡流水
func (feishu *Feishu) UserFlowsDelete(recordIds ...string) (successRecordIds []string, failRecordIds []string, err error) {
	req := larkattendance.NewBatchDelUserFlowReqBuilder().
		Body(larkattendance.NewBatchDelUserFlowReqBodyBuilder().RecordIds(recordIds).Build()).
		Build()

	var resp *larkattendance.BatchDelUserFlowResp
	resp, err = parseResponse(feishu.client.Attendance.V1.UserFlow.BatchDel(context.Background(), req))
	if err != nil {
		return
	}
	return resp.Data.SuccessRecordIds, resp.Data.FailRecordIds, nil
}
