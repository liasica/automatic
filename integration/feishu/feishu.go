// Copyright (C) automatic. 2026-present.
//
// Created at 2026-02-22, by liasica

package feishu

import (
	"errors"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"

	"automatic/core"
)

type Feishu struct {
	client *lark.Client
}

func New(cfg *core.Config, cache *core.Cache) *Feishu {
	return &Feishu{
		client: lark.NewClient(
			cfg.Lark.AppId,
			cfg.Lark.AppSecret,
			lark.WithLogReqAtDebug(true),
			lark.WithLogger(&Logger{}),
			lark.WithLogLevel(larkcore.LogLevelDebug),
			lark.WithTokenCache(cache),
			lark.WithEnableTokenCache(true),
		),
	}
}

type Response interface {
	Success() bool
	ErrorResp() string
}

func parseResponse[T Response](resp T, err error) (T, error) {
	var zero T
	if err != nil {
		return zero, err
	}

	if !resp.Success() {
		return zero, errors.New(resp.ErrorResp())
	}

	return resp, nil
}
