// Copyright (C) automatic. 2026-present.
//
// Created at 2026-04-15, by liasica

package handler

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"automatic/core"
	"automatic/integration/feishu"
)

// Attendance 打卡记录处理器
type Attendance struct {
	feishu *feishu.Feishu
	config *core.Config
}

// NewAttendance 创建打卡记录处理器
func NewAttendance(fs *feishu.Feishu, cfg *core.Config) *Attendance {
	return &Attendance{feishu: fs, config: cfg}
}

const (
	dateFormat     = "2006-01-02"
	dateTimeFormat = "2006-01-02 15:04:05"
)

// userInfo 用户信息响应
type userInfo struct {
	ID              string `json:"id"`
	CheckInLatest   string `json:"check_in_latest"`
	CheckOutEarliest string `json:"check_out_earliest"`
}

// attendanceRecord 打卡记录响应
type attendanceRecord struct {
	RecordID  string `json:"record_id"`
	UserID    string `json:"user_id"`
	CheckTime string `json:"check_time"`
	Timestamp int64  `json:"timestamp"`
	Label     string `json:"label"`
	DayType   string `json:"day_type"`
	Weekday   string `json:"weekday"`
}

// createRequest 添加打卡记录请求
type createRequest struct {
	UserID    string `json:"user_id"`
	CheckTime string `json:"check_time"`
}

// deleteRequest 删除打卡记录请求
type deleteRequest struct {
	RecordIDs []string `json:"record_ids"`
}

// Users 返回配置的用户列表
func (a *Attendance) Users(c echo.Context) error {
	users := make([]userInfo, 0, len(a.config.Users))
	for _, u := range a.config.Users {
		users = append(users, userInfo{
			ID:              u.Id,
			CheckInLatest:   u.CheckIn.Latest,
			CheckOutEarliest: u.CheckOut.Earliest,
		})
	}
	return c.JSON(http.StatusOK, echo.Map{"users": users})
}

// Query 查询打卡记录，支持按用户分组和标签
func (a *Attendance) Query(c echo.Context) error {
	fromStr := c.QueryParam("from")
	if fromStr == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "from 参数必填"})
	}

	from, err := time.ParseInLocation(dateFormat, fromStr, time.Local)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "from 日期格式无效，请使用 YYYY-MM-DD"})
	}

	var to time.Time
	toStr := c.QueryParam("to")
	if toStr == "" {
		to = time.Date(from.Year(), from.Month(), from.Day(), 23, 59, 59, 0, time.Local)
	} else {
		to, err = time.ParseInLocation(dateFormat, toStr, time.Local)
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "to 日期格式无效，请使用 YYYY-MM-DD"})
		}
		to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, time.Local)
	}

	// 确定查询用户范围
	userID := c.QueryParam("user_id")
	var targetUsers []*core.User
	if userID == "" || userID == "all" {
		targetUsers = a.config.Users
	} else {
		for _, u := range a.config.Users {
			if u.Id == userID {
				targetUsers = []*core.User{u}
				break
			}
		}
		if targetUsers == nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "未找到该用户"})
		}
	}

	// 按用户分组查询
	type userGroup struct {
		UserID  string             `json:"user_id"`
		Records []attendanceRecord `json:"records"`
	}

	groups := make([]userGroup, 0, len(targetUsers))

	for _, user := range targetUsers {
		data, queryErr := a.feishu.UserFlowsQuery(user.Id, from, to)
		if queryErr != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": queryErr.Error()})
		}

		records := make([]attendanceRecord, 0)
		if data != nil {
			for _, flow := range data.UserFlowResults {
				if flow == nil || flow.RecordId == nil || flow.CheckTime == nil {
					continue
				}

				ts, parseErr := strconv.ParseInt(*flow.CheckTime, 10, 64)
				if parseErr != nil {
					continue
				}

				t := time.Unix(ts, 0).In(time.Local)

				records = append(records, attendanceRecord{
					RecordID:  *flow.RecordId,
					UserID:    user.Id,
					CheckTime: t.Format(dateTimeFormat),
					Timestamp: ts,
					Label:     a.detectLabel(user, t),
					DayType:   core.GetDayType(t),
					Weekday:   weekdayCN(t.Weekday()),
				})
			}
		}

		// 时间倒序
		sort.Slice(records, func(i, j int) bool {
			return records[i].Timestamp > records[j].Timestamp
		})

		groups = append(groups, userGroup{
			UserID:  user.Id,
			Records: records,
		})
	}

	return c.JSON(http.StatusOK, echo.Map{"groups": groups})
}

// detectLabel 根据用户配置和打卡时间判断上下班标签
func (a *Attendance) detectLabel(user *core.User, t time.Time) string {
	checkInDeadline := time.Date(t.Year(), t.Month(), t.Day(),
		user.CheckIn.LatestTime.Hour, user.CheckIn.LatestTime.Minute, 0, 0, time.Local)
	checkOutStart := time.Date(t.Year(), t.Month(), t.Day(),
		user.CheckOut.EarliestTime.Hour, user.CheckOut.EarliestTime.Minute, 0, 0, time.Local)

	// 中间分界线取两个时间的中点
	mid := checkInDeadline.Add(checkOutStart.Sub(checkInDeadline) / 2)

	if t.Before(mid) {
		return "checkin"
	}
	return "checkout"
}

// weekdayCN 返回中文星期
func weekdayCN(wd time.Weekday) string {
	names := [...]string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	return names[wd]
}

// Create 添加打卡记录
func (a *Attendance) Create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "请求体格式无效"})
	}

	if req.UserID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "user_id 参数必填"})
	}
	if req.CheckTime == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "check_time 参数必填"})
	}

	checkTime, err := time.ParseInLocation(dateTimeFormat, req.CheckTime, time.Local)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "check_time 格式无效，请使用 YYYY-MM-DD HH:MM:SS"})
	}

	err = a.feishu.UserFlowsCreate(req.UserID, checkTime)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	punchType := feishu.PunchTypeCheckIn
	if checkTime.Hour() >= 12 {
		punchType = feishu.PunchTypeCheckOut
	}
	if notifyErr := a.feishu.NotifyPunchSuccess(req.UserID, punchType, checkTime, feishu.PunchSourceManual); notifyErr != nil {
		// 通知失败不阻塞打卡结果
		c.Logger().Warnf("手动添加打卡成功通知发送失败，user: %s, error: %v", req.UserID, notifyErr)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "打卡记录添加成功"})
}

// Delete 删除打卡记录
func (a *Attendance) Delete(c echo.Context) error {
	var req deleteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "请求体格式无效"})
	}

	if len(req.RecordIDs) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "record_ids 参数必填"})
	}

	successIDs, failIDs, err := a.feishu.UserFlowsDelete(req.RecordIDs...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"success_ids": successIDs,
		"fail_ids":    failIDs,
	})
}
