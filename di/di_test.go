// Copyright (C) automatic. 2026-present.
//
// Created at 2026-02-26, by liasica

package di

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
	"go.uber.org/fx"

	"automatic/core"
	"automatic/integration/feishu"
	"automatic/integration/openwrt"
)

func TestNew(t *testing.T) {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Value: "../configs/config.yaml",
			},
		},
	}
	err := New(cmd,
		fx.Invoke(func(cfg *core.Config, fs *feishu.Feishu, ow *openwrt.OpenWrt) {
			require.NotNil(t, cfg)
			require.NotNil(t, fs)
			require.NotNil(t, ow)
		}),
	).Start(context.Background())
	require.NoError(t, err)
}
