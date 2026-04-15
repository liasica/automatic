// Copyright (C) automatic. 2026-present.
//
// Created at 2026-04-15, by liasica

package feishu

import (
	"context"
	"fmt"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	larkext "github.com/larksuite/oapi-sdk-go/v3/service/ext"
)

// ExchangeUserToken 用授权码换取 user_access_token
func (feishu *Feishu) ExchangeUserToken(code string) (string, error) {
	req := larkext.NewAuthenAccessTokenReqBuilder().
		Body(larkext.NewAuthenAccessTokenReqBodyBuilder().
			GrantType("authorization_code").
			Code(code).
			Build()).
		Build()

	resp, err := feishu.client.Ext.Authen.AuthenAccessToken(context.Background(), req)
	if err != nil {
		return "", err
	}
	if !resp.Success() {
		return "", fmt.Errorf("获取 user_access_token 失败: code=%d msg=%s", resp.Code, resp.Msg)
	}

	return resp.Data.AccessToken, nil
}

// GrantBitableAccess 使用 user_access_token 给应用授予多维表格编辑权限
func (feishu *Feishu) GrantBitableAccess(userToken, appToken string) error {
	// 先获取当前用户信息（即应用的身份信息不同，这里用 user token 获取当前用户 open_id）
	// 然后将应用添加为协作者
	// 对于自建应用，使用 user_access_token 设置文档链接分享为组织内可编辑
	req := larkdrive.NewPatchPermissionPublicReqBuilder().
		Token(appToken).
		Type("bitable").
		PermissionPublicRequest(larkdrive.NewPermissionPublicRequestBuilder().
			LinkShareEntity("tenant_editable").
			Build()).
		Build()

	resp, err := feishu.client.Drive.V1.PermissionPublic.Patch(
		context.Background(), req,
		larkcore.WithUserAccessToken(userToken),
	)
	if err != nil {
		return err
	}
	if !resp.Success() {
		return fmt.Errorf("授权失败: code=%d msg=%s", resp.Code, resp.Msg)
	}

	return nil
}
