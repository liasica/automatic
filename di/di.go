// Copyright (C) automatic. 2026-present.
//
// Created at 2026-02-26, by liasica

package di

import (
	"github.com/urfave/cli/v3"
	"go.uber.org/fx"

	"automatic/core"
	"automatic/integration/feishu"
	"automatic/integration/openwrt"
)

func New(cmd *cli.Command, opts ...fx.Option) *fx.App {
	return fx.New(
		// 全局 Providers
		fx.Provide(
			// 加载配置
			func() (*core.Config, error) {
				return core.NewConfig(cmd.String("config"))
			},

			// 加载缓存
			func(cfg *core.Config) (*core.Cache, error) {
				return core.NewCache(cfg.Redis.Addr, cfg.Redis.DB)
			},

			// integrations
			// 加载 feishu
			feishu.New,

			// 加载 openwrt
			openwrt.New,
		),

		// 合并选项
		fx.Options(opts...),
	)
}
