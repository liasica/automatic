// Copyright (C) automatic. 2026-present.
//
// Created at 2026-02-26, by liasica

package core

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

func NewCache(addr string, db int) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   db,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &Cache{
		client: client,
	}, nil
}

func (c *Cache) Redis() *redis.Client {
	return c.client
}

func (c *Cache) Set(ctx context.Context, key string, value string, expireTime time.Duration) error {
	return c.client.Set(ctx, key, value, expireTime).Err()
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	value, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return value, err
}

func (c *Cache) SetIfAbsent(ctx context.Context, key string, value string, expireTime time.Duration) (bool, error) {
	cmd := c.client.SetArgs(ctx, key, value, redis.SetArgs{
		Mode: "NX",
		TTL:  expireTime,
	})
	if err := cmd.Err(); err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *Cache) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *Cache) LPush(ctx context.Context, key string, values ...string) error {
	if len(values) == 0 {
		return nil
	}
	args := make([]interface{}, 0, len(values))
	for _, value := range values {
		args = append(args, value)
	}
	return c.client.LPush(ctx, key, args...).Err()
}

func (c *Cache) LMoveRightToLeft(ctx context.Context, source string, destination string) (string, error) {
	return c.client.LMove(ctx, source, destination, "RIGHT", "LEFT").Result()
}

func (c *Cache) LRem(ctx context.Context, key string, count int64, value string) (int64, error) {
	return c.client.LRem(ctx, key, count, value).Result()
}

func (c *Cache) LRange(ctx context.Context, key string, start int64, stop int64) ([]string, error) {
	return c.client.LRange(ctx, key, start, stop).Result()
}

func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}
