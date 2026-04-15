// Copyright (C) automatic. 2026-present.
//
// Created at 2026-04-15, by liasica

package feishu

import (
	"context"

	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
)

// BitableListTables 获取多维表格的所有数据表
func (feishu *Feishu) BitableListTables(appToken string) ([]*larkbitable.AppTable, error) {
	req := larkbitable.NewListAppTableReqBuilder().
		AppToken(appToken).
		PageSize(100).
		Build()

	resp, err := parseResponse(feishu.client.Bitable.V1.AppTable.List(context.Background(), req))
	if err != nil {
		return nil, err
	}

	return resp.Data.Items, nil
}

// BitableListFields 获取数据表的字段列表
func (feishu *Feishu) BitableListFields(appToken, tableId string) ([]*larkbitable.AppTableFieldForList, error) {
	req := larkbitable.NewListAppTableFieldReqBuilder().
		AppToken(appToken).
		TableId(tableId).
		PageSize(100).
		Build()

	resp, err := parseResponse(feishu.client.Bitable.V1.AppTableField.List(context.Background(), req))
	if err != nil {
		return nil, err
	}

	return resp.Data.Items, nil
}

// BitableListRecords 查询数据表记录
func (feishu *Feishu) BitableListRecords(appToken, tableId string, filter string, pageSize int) ([]*larkbitable.AppTableRecord, error) {
	builder := larkbitable.NewListAppTableRecordReqBuilder().
		AppToken(appToken).
		TableId(tableId).
		PageSize(pageSize)

	if filter != "" {
		builder.Filter(filter)
	}

	var allRecords []*larkbitable.AppTableRecord
	pageToken := ""

	for {
		if pageToken != "" {
			builder.PageToken(pageToken)
		}

		resp, err := parseResponse(feishu.client.Bitable.V1.AppTableRecord.List(context.Background(), builder.Build()))
		if err != nil {
			return nil, err
		}

		allRecords = append(allRecords, resp.Data.Items...)

		if resp.Data.HasMore == nil || !*resp.Data.HasMore {
			break
		}
		if resp.Data.PageToken != nil {
			pageToken = *resp.Data.PageToken
		}
	}

	return allRecords, nil
}

// BitableUpdateRecord 更新数据表记录
func (feishu *Feishu) BitableUpdateRecord(appToken, tableId, recordId string, fields map[string]any) (*larkbitable.AppTableRecord, error) {
	record := larkbitable.NewAppTableRecordBuilder().
		Fields(fields).
		Build()

	req := larkbitable.NewUpdateAppTableRecordReqBuilder().
		AppToken(appToken).
		TableId(tableId).
		RecordId(recordId).
		AppTableRecord(record).
		Build()

	resp, err := parseResponse(feishu.client.Bitable.V1.AppTableRecord.Update(context.Background(), req))
	if err != nil {
		return nil, err
	}

	return resp.Data.Record, nil
}

// BitableCreateRecord 创建数据表记录
func (feishu *Feishu) BitableCreateRecord(appToken, tableId string, fields map[string]any) (*larkbitable.AppTableRecord, error) {
	record := larkbitable.NewAppTableRecordBuilder().
		Fields(fields).
		Build()

	req := larkbitable.NewCreateAppTableRecordReqBuilder().
		AppToken(appToken).
		TableId(tableId).
		AppTableRecord(record).
		Build()

	resp, err := parseResponse(feishu.client.Bitable.V1.AppTableRecord.Create(context.Background(), req))
	if err != nil {
		return nil, err
	}

	return resp.Data.Record, nil
}
