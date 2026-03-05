// Copyright (C) automatic. 2026-present.
//
// Created at 2026-02-27, by liasica

package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseTime(t *testing.T) {
	tt, err := time.Parse("15:04", "6:30")
	require.NoError(t, err)
	t.Logf("%v", tt)
}
