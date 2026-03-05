// Copyright (C) automatic. 2026-present.
//
// Created at 2026-02-26, by liasica

package openwrt

import (
	"testing"

	"github.com/stretchr/testify/require"

	"automatic/core"
)

func getConfig(t *testing.T) *core.Config {
	cfg, err := core.NewConfig("../../configs/config.yaml")
	require.NoError(t, err)

	return cfg
}

func TestGetOnlineDevices(t *testing.T) {
	devices, err := New(getConfig(t)).GetDevices()
	require.NoError(t, err)

	for _, device := range devices {
		t.Logf("Device: %s, IP: %s, FLAGS: %s", device.Mac, device.Ip, device.Flags)
	}
}
