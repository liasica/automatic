// Copyright (C) automatic. 2026-present.
//
// Created at 2026-04-15, by liasica

package command

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"automatic/core"
	"automatic/integration/feishu"
	"automatic/service"
	"automatic/service/handler"
)

// registerHTTPServer 在指定的 fx.Lifecycle 上挂载一个 Web Dashboard HTTP 服务
// addr 为空时不启动，便于 `punch run` 通过 flag 控制是否集成 Dashboard
func registerHTTPServer(lc fx.Lifecycle, addr string, cfg *core.Config, fs *feishu.Feishu) {
	if addr == "" {
		return
	}

	attendance := handler.NewAttendance(fs, cfg)
	overtime := handler.NewOvertime(fs, cfg)
	auth := handler.NewAuth(fs, cfg)
	srv := service.NewServer(addr, attendance, overtime, auth)

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := srv.Start(); err != nil {
					zap.S().Infof("HTTP 服务器已关闭: %v", err)
				}
			}()
			zap.S().Infof("Web Dashboard 服务已启动，监听地址: %s", addr)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			zap.S().Info("正在关闭 Web Dashboard 服务")
			return srv.Shutdown(ctx)
		},
	})
}
