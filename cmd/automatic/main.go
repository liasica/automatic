// Copyright (C) automatic. 2026-present.
//
// Created at 2026-02-22, by liasica

package main

import (
	"context"
	"os"
	"time"

	"github.com/urfave/cli/v3"
	"go.uber.org/zap"

	"automatic/cmd/automatic/internal/command"
)

// 定义全局 flags
var globalFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Value:   "./configs/config.yaml",
		Usage:   "配置文件路径, 例如: ./configs/config.yaml",
		Sources: cli.EnvVars("CONFIG"), // 支持环境变量
	},
}

// initConfigHook 初始化配置的 Before 钩子
var initConfigHook = func(ctx context.Context, c *cli.Command) (context.Context, error) {
	return ctx, nil
}

// addGlobalFlags 为命令添加全局 flags 和配置初始化钩子
func addGlobalFlags(cmd *cli.Command) *cli.Command {
	cmd.Flags = append(globalFlags, cmd.Flags...)
	// 为子命令添加 Before 钩子以初始化配置
	if cmd.Before == nil {
		cmd.Before = initConfigHook
	}
	return cmd
}

func main() {
	// 设置全局时区
	tz := "Asia/Shanghai"
	_ = os.Setenv("TZ", tz)
	loc, _ := time.LoadLocation(tz)
	time.Local = loc

	l, _ := zap.NewDevelopment(zap.WithCaller(true), zap.AddStacktrace(zap.DPanicLevel))
	zap.ReplaceGlobals(l)

	app := &cli.Command{
		Name:    "automatic",
		Usage:   "自动化工具，实现如：自动打卡等辅助工作的工具",
		Version: "0.1.0",
		Flags:   globalFlags,

		// 子命令
		Commands: []*cli.Command{
			addGlobalFlags(command.NewPunch().Command),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		zap.S().Errorf("应用运行失败：%v", err)
		os.Exit(1)
	}
}
