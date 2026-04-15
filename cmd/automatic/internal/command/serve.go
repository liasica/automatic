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
)

// Serve Web Dashboard 命令
type Serve struct {
	*cli.Command
}

// NewServe 创建 Serve 命令
// 该命令只启动 Web Dashboard，不涉及自动打卡；适用于仅前端联调场景
// 生产部署建议使用 `punch run`，打卡与 Dashboard 合并在同一进程
func NewServe() (s *Serve) {
	s = &Serve{}
	s.Command = &cli.Command{
		Name:        "serve",
		Usage:       "仅启动 Web Dashboard 服务（不含自动打卡）",
		Description: "仅启动 HTTP 服务器提供 REST API，用于前端开发联调；生产环境请使用 punch run",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "addr",
				Aliases: []string{"a"},
				Value:   ":9876",
				Usage:   "HTTP 监听地址",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			addr := cmd.String("addr")

			app := di.New(cmd, fx.Invoke(func(lc fx.Lifecycle, cfg *core.Config, fs *feishu.Feishu) {
				registerHTTPServer(lc, addr, cfg, fs)
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
