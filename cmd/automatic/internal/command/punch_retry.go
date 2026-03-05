// Copyright (C) automatic. 2026-present.
//
// Created at 2026-03-05, by codex

package command

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"automatic/core"
	"automatic/integration/feishu"
)

const (
	// 待重试失败请求队列
	failedPunchPendingKey = "automatic:punch:failed:pending"
	// 重试处理中队列，用于在进程中断后恢复
	failedPunchProcessingKey = "automatic:punch:failed:processing"
	// 无法解析的脏数据队列
	failedPunchInvalidKey = "automatic:punch:failed:invalid"
	// 分布式重试锁，避免并发重复消费
	failedPunchRetryLockKey = "automatic:punch:failed:retry:lock"
	defaultRetryInterval    = 10 * time.Minute
)

// failedPunchRequest 表示一次失败打卡请求的最小重试上下文
type failedPunchRequest struct {
	RequestID       string `json:"requestId"`
	PunchType       string `json:"punchType"`
	UserID          string `json:"userId"`
	Mac             string `json:"mac,omitempty"`
	CheckTimeUnix   int64  `json:"checkTimeUnix"`
	Source          string `json:"source"`
	CreatedAtUnix   int64  `json:"createdAtUnix"`
	RetryCount      int    `json:"retryCount"`
	LastRetryAtUnix int64  `json:"lastRetryAtUnix,omitempty"`
	LastError       string `json:"lastError,omitempty"`
}

type retryFailedResult struct {
	Total   int
	Success int
	Failed  int
	Invalid int
}

// failedPunchRetryInterval 返回配置的重试间隔，缺省回退 10m
func (p *Punch) failedPunchRetryInterval(cfg *core.Config) time.Duration {
	if cfg == nil || cfg.Retry == nil || cfg.Retry.FailedPunchIntervalDuration <= 0 {
		return defaultRetryInterval
	}
	return cfg.Retry.FailedPunchIntervalDuration
}

// newFailedPunchRequest 构造失败请求记录
func (p *Punch) newFailedPunchRequest(punchType string, userID string, mac string, checkTime time.Time, source string, cause error) *failedPunchRequest {
	return &failedPunchRequest{
		RequestID:     fmt.Sprintf("%s:%d:%d", userID, checkTime.Unix(), time.Now().UnixNano()),
		PunchType:     punchType,
		UserID:        userID,
		Mac:           mac,
		CheckTimeUnix: checkTime.Unix(),
		Source:        source,
		CreatedAtUnix: time.Now().Unix(),
		LastError:     cause.Error(),
	}
}

// recordFailedPunch 将失败请求写入待重试队列
func (p *Punch) recordFailedPunch(ctx context.Context, cache *core.Cache, req *failedPunchRequest) error {
	payload, err := sonic.Marshal(req)
	if err != nil {
		return err
	}
	return cache.LPush(ctx, failedPunchPendingKey, string(payload))
}

// recoverProcessingQueue 将上次中断遗留的处理中数据回收到待重试队列
func (p *Punch) recoverProcessingQueue(ctx context.Context, cache *core.Cache) (int, error) {
	items, err := cache.LRange(ctx, failedPunchProcessingKey, 0, -1)
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}
	if err = cache.LPush(ctx, failedPunchPendingKey, items...); err != nil {
		return 0, err
	}
	if err = cache.Del(ctx, failedPunchProcessingKey); err != nil {
		return 0, err
	}
	return len(items), nil
}

// retryFailedPunches 执行一轮失败打卡重试
// 队列消费模型: pending(RIGHT) -> processing(LEFT), 成功/失败后从 processing 删除
func (p *Punch) retryFailedPunches(ctx context.Context, cfg *core.Config, cache *core.Cache, fs *feishu.Feishu, trigger string) (result retryFailedResult, err error) {
	lockTTL := p.failedPunchRetryInterval(cfg)
	acquired, err := cache.SetIfAbsent(ctx, failedPunchRetryLockKey, strconv.FormatInt(time.Now().UnixNano(), 10), lockTTL)
	if err != nil {
		return result, fmt.Errorf("抢占失败打卡重试锁失败: %w", err)
	}
	if !acquired {
		zap.S().Debugf("失败打卡重试任务跳过，已有任务执行中，trigger: %s", trigger)
		return result, nil
	}
	defer func() {
		if delErr := cache.Del(ctx, failedPunchRetryLockKey); delErr != nil {
			zap.S().Errorf("释放失败打卡重试锁失败，error: %v", delErr)
		}
	}()

	recovered, err := p.recoverProcessingQueue(ctx, cache)
	if err != nil {
		return result, fmt.Errorf("恢复失败打卡处理中队列失败: %w", err)
	}
	if recovered > 0 {
		zap.S().Warnf("失败打卡重试发现处理中遗留数据，已回收到待重试队列，count: %d", recovered)
	}

	for {
		var payload string
		// 原子转移一条消息到 processing，防止进程中断时消息丢失
		payload, err = cache.LMoveRightToLeft(ctx, failedPunchPendingKey, failedPunchProcessingKey)
		if err != nil {
			if errors.Is(err, redis.Nil) {
				err = nil
				break
			}
			return result, fmt.Errorf("拉取失败打卡记录失败: %w", err)
		}
		result.Total++

		record := &failedPunchRequest{}
		if unmarshalErr := sonic.UnmarshalString(payload, record); unmarshalErr != nil {
			result.Invalid++
			_, _ = cache.LRem(ctx, failedPunchProcessingKey, 1, payload)
			_ = cache.LPush(ctx, failedPunchInvalidKey, payload)
			zap.S().Errorf("解析失败打卡记录失败，已移入无效队列，payload: %s, error: %v", payload, unmarshalErr)
			continue
		}

		checkTime := time.Unix(record.CheckTimeUnix, 0)
		if submitErr := fs.UserFlowsCreate(record.UserID, checkTime); submitErr != nil {
			result.Failed++
			record.RetryCount++
			record.LastRetryAtUnix = time.Now().Unix()
			record.LastError = submitErr.Error()

			updatedPayload := payload
			if updatedBytes, marshalErr := sonic.Marshal(record); marshalErr == nil {
				updatedPayload = string(updatedBytes)
			} else {
				zap.S().Errorf("序列化失败打卡记录失败，requestId: %s, error: %v", record.RequestID, marshalErr)
			}

			// 提交失败后回写到 pending，形成后续重试
			if pushErr := cache.LPush(ctx, failedPunchPendingKey, updatedPayload); pushErr != nil {
				return result, fmt.Errorf("回写失败打卡记录失败: %w", pushErr)
			}
			if _, remErr := cache.LRem(ctx, failedPunchProcessingKey, 1, payload); remErr != nil {
				return result, fmt.Errorf("清理处理中失败打卡记录失败: %w", remErr)
			}
			zap.S().Warnf("重试失败打卡失败，requestId: %s, user: %s, punchType: %s, retryCount: %d, error: %v", record.RequestID, record.UserID, record.PunchType, record.RetryCount, submitErr)
			continue
		}

		result.Success++
		if _, remErr := cache.LRem(ctx, failedPunchProcessingKey, 1, payload); remErr != nil {
			return result, fmt.Errorf("清理已成功的处理中打卡记录失败: %w", remErr)
		}
		zap.S().Infof("重试失败打卡成功，requestId: %s, user: %s, punchType: %s, retryCount: %d", record.RequestID, record.UserID, record.PunchType, record.RetryCount)
	}
	return result, nil
}
