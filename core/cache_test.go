// Copyright (C) automatic. 2026-present.
//
// Created at 2026-02-27, by liasica

package core

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewCache(t *testing.T) {
	cache, err := NewCache("localhost:6379", 0)
	require.NoError(t, err)

	ctx := context.Background()
	err = cache.Redis().HSet(ctx, "test_key", "field1", "value1").Err()
	require.NoError(t, err)

	err = cache.Redis().Expire(ctx, "test_key", 1*time.Second).Err()
	require.NoError(t, err)

	m := cache.Redis().HGetAll(ctx, "test_key").Val()
	require.NotNil(t, m)
	require.Equal(t, "value1", m["field1"])

	time.Sleep(1 * time.Second)

	m = cache.Redis().HGetAll(ctx, "test_key").Val()
	require.Empty(t, m)
}
