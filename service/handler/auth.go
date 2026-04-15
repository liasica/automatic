// Copyright (C) automatic. 2026-present.
//
// Created at 2026-04-15, by liasica

package handler

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"automatic/core"
	"automatic/integration/feishu"
)

// Auth 飞书授权处理器
type Auth struct {
	feishu      *feishu.Feishu
	appId       string
	appToken    string // 加班记录多维表格 app_token
	redirectURL string // OAuth 回调 URL，优先使用配置
}

// NewAuth 创建授权处理器
func NewAuth(fs *feishu.Feishu, cfg *core.Config) *Auth {
	return &Auth{
		feishu:      fs,
		appId:       cfg.Lark.AppId,
		appToken:    cfg.Lark.OvertimeAppToken,
		redirectURL: cfg.Lark.RedirectURL,
	}
}

// Redirect 重定向到飞书 OAuth 授权页
func (a *Auth) Redirect(c echo.Context) error {
	redirectURI := a.redirectURL
	if redirectURI == "" {
		scheme := c.Scheme()
		if fwd := c.Request().Header.Get("X-Forwarded-Proto"); fwd != "" {
			scheme = fwd
		}
		redirectURI = fmt.Sprintf("%s://%s/api/auth/feishu/callback", scheme, c.Request().Host)
	}

	// 打印实际使用的 redirect_uri，便于在飞书开发者后台「安全设置 → 重定向URL」添加白名单
	zap.S().Infof("飞书 OAuth 使用 redirect_uri: %s", redirectURI)

	// 申请用户级权限 scope：
	// docs:permission.setting:write_only 用于 Drive PermissionPublic.Patch 调整链接分享
	// （bitable:bitable 是套件级 scope，仅对 ISV 应用开放，自建应用无法申请）
	scope := "docs:permission.setting:write_only"

	authURL := fmt.Sprintf(
		"https://open.feishu.cn/open-apis/authen/v1/authorize?app_id=%s&redirect_uri=%s&scope=%s&state=grant_bitable",
		a.appId,
		url.QueryEscape(redirectURI),
		url.QueryEscape(scope),
	)

	return c.Redirect(http.StatusFound, authURL)
}

// Callback 处理飞书 OAuth 回调，使用用户令牌授权多维表格
func (a *Auth) Callback(c echo.Context) error {
	code := c.QueryParam("code")
	if code == "" {
		return c.HTML(http.StatusBadRequest, page("授权失败", "未获取到授权码", false))
	}

	if a.appToken == "" {
		return c.HTML(http.StatusBadRequest, page("授权失败", "未配置加班记录多维表格 OvertimeAppToken", false))
	}

	// 用授权码换取 user_access_token
	userToken, err := a.feishu.ExchangeUserToken(code)
	if err != nil {
		return c.HTML(http.StatusInternalServerError, page("授权失败", err.Error(), false))
	}

	// 使用用户令牌设置多维表格链接分享为组织内可编辑
	err = a.feishu.GrantBitableAccess(userToken, a.appToken)
	if err != nil {
		return c.HTML(http.StatusInternalServerError, page("授权失败", err.Error(), false))
	}

	return c.HTML(http.StatusOK, page("授权成功", "多维表格已设置为组织内可编辑，即将返回首页…", true))
}

// page 渲染简单状态页，autoRedirect 为 true 时 2 秒后跳回首页
func page(title, message string, autoRedirect bool) string {
	redirect := ""
	if autoRedirect {
		redirect = `<meta http-equiv="refresh" content="2;url=/">`
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>%s</title>%s
<style>body{font-family:system-ui;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0;background:#faf9f6}
.card{background:#fff;border:1px solid #dedbd6;border-radius:8px;padding:32px;max-width:400px;text-align:center}
h1{font-size:18px;margin:0 0 8px}p{color:#706e6b;font-size:14px;margin:0 0 12px}a{color:#2d6cdf;font-size:13px;text-decoration:none}a:hover{text-decoration:underline}</style>
</head><body><div class="card"><h1>%s</h1><p>%s</p><a href="/">返回首页</a></div></body></html>`, title, redirect, title, message)
}
