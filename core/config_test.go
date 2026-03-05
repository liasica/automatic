// Copyright (C) automatic. 2026-present.
//
// Created at 2026-02-26, by liasica

package core

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	cfg, err := NewConfig("../configs/config.yaml")
	require.NoError(t, err)
	t.Logf("%#v", cfg)
}
