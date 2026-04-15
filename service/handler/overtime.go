// Copyright (C) automatic. 2026-present.
//
// Created at 2026-04-15, by liasica

package handler

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"

	"automatic/core"
	"automatic/integration/feishu"
)

// Overtime 加班记录处理器
type Overtime struct {
	feishu   *feishu.Feishu
	appToken string

	mu      sync.Mutex
	tableId string // 懒加载缓存
}

// NewOvertime 创建加班记录处理器
func NewOvertime(fs *feishu.Feishu, cfg *core.Config) *Overtime {
	return &Overtime{
		feishu:   fs,
		appToken: cfg.Lark.OvertimeAppToken,
	}
}

// ensureTableId 确保已获取 tableId
func (o *Overtime) ensureTableId() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.tableId != "" {
		return nil
	}

	tables, err := o.feishu.BitableListTables(o.appToken)
	if err != nil {
		return fmt.Errorf("获取数据表列表失败: %w", err)
	}
	if len(tables) == 0 {
		return fmt.Errorf("多维表格中没有数据表")
	}

	o.tableId = *tables[0].TableId
	return nil
}

// schemaField 字段信息响应
type schemaField struct {
	FieldId   string `json:"field_id"`
	FieldName string `json:"field_name"`
	Type      int    `json:"type"`
	UiType    string `json:"ui_type,omitempty"`
}

// Schema 返回数据表字段结构
func (o *Overtime) Schema(c echo.Context) error {
	if o.appToken == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "未配置加班记录多维表格"})
	}

	if err := o.ensureTableId(); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	fields, err := o.feishu.BitableListFields(o.appToken, o.tableId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	result := make([]schemaField, 0, len(fields))
	for _, f := range fields {
		sf := schemaField{}
		if f.FieldId != nil {
			sf.FieldId = *f.FieldId
		}
		if f.FieldName != nil {
			sf.FieldName = *f.FieldName
		}
		if f.Type != nil {
			sf.Type = *f.Type
		}
		if f.UiType != nil {
			sf.UiType = *f.UiType
		}
		result = append(result, sf)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"table_id": o.tableId,
		"fields":   result,
	})
}

// overtimeRecord 加班记录响应
type overtimeRecord struct {
	RecordId string         `json:"record_id"`
	Fields   map[string]any `json:"fields"`
}

// Query 查询加班记录
func (o *Overtime) Query(c echo.Context) error {
	if o.appToken == "" {
		return c.JSON(http.StatusOK, echo.Map{"records": []overtimeRecord{}})
	}

	if err := o.ensureTableId(); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	filter := c.QueryParam("filter")

	records, err := o.feishu.BitableListRecords(o.appToken, o.tableId, filter, 500)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	result := make([]overtimeRecord, 0, len(records))
	for _, r := range records {
		if r.RecordId == nil {
			continue
		}
		result = append(result, overtimeRecord{
			RecordId: *r.RecordId,
			Fields:   r.Fields,
		})
	}

	return c.JSON(http.StatusOK, echo.Map{"records": result})
}

// updateRequest 更新请求
type updateRequest struct {
	Fields map[string]any `json:"fields"`
}

// Update 更新加班记录
func (o *Overtime) Update(c echo.Context) error {
	if o.appToken == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "未配置加班记录多维表格"})
	}

	recordId := c.Param("record_id")
	if recordId == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "record_id 参数必填"})
	}

	var req updateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "请求体格式无效"})
	}

	if len(req.Fields) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "fields 不能为空"})
	}

	if err := o.ensureTableId(); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	record, err := o.feishu.BitableUpdateRecord(o.appToken, o.tableId, recordId, req.Fields)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	result := overtimeRecord{Fields: record.Fields}
	if record.RecordId != nil {
		result.RecordId = *record.RecordId
	}

	return c.JSON(http.StatusOK, echo.Map{"record": result})
}

// createOvertimeRequest 创建请求
type createOvertimeRequest struct {
	Fields map[string]any `json:"fields"`
}

// Create 创建加班记录
func (o *Overtime) Create(c echo.Context) error {
	if o.appToken == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "未配置加班记录多维表格"})
	}

	var req createOvertimeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "请求体格式无效"})
	}

	if len(req.Fields) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "fields 不能为空"})
	}

	if err := o.ensureTableId(); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	record, err := o.feishu.BitableCreateRecord(o.appToken, o.tableId, req.Fields)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	result := overtimeRecord{Fields: record.Fields}
	if record.RecordId != nil {
		result.RecordId = *record.RecordId
	}

	return c.JSON(http.StatusOK, echo.Map{"record": result})
}
