// Copyright (C) automatic. 2026-present.
//
// Created at 2026-04-15, by liasica

package command

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"automatic/core"
	"automatic/di"
	"automatic/integration/feishu"
	"automatic/service"
	"automatic/service/handler"
)

// Serve Web Dashboard 命令
type Serve struct {
	*cli.Command
}

// NewServe 创建 Serve 命令
func NewServe() (s *Serve) {
	s = &Serve{}
	s.Command = &cli.Command{
		Name:        "serve",
		Usage:       "启动 Web Dashboard 服务",
		Description: "启动 HTTP 服务器，提供打卡记录的 REST API",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "addr",
				Aliases: []string{"a"},
				Value:   ":8080",
				Usage:   "HTTP 监听地址",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			addr := cmd.String("addr")

			app := di.New(cmd, fx.Invoke(func(lc fx.Lifecycle, cfg *core.Config, fs *feishu.Feishu) {
				attendance := handler.NewAttendance(fs, cfg)
				srv := service.NewServer(addr, attendance)

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
			}))

			if err := app.Start(ctx); err != nil {
				return err
			}

			ch := make(chan os.Signal, 1)
			signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(ch)

			select {
			case <-ctx.Done():
			case sig := <-ch:
				zap.S().Infof("收到退出信号: %s", sig.String())
			}

			stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return app.Stop(stopCtx)
		},
	}
	return
}
