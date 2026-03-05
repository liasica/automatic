// Copyright (C) automatic. 2026-present.
//
// Created at 2026-02-25, by liasica

package feishu

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type Logger struct {
}

func (l *Logger) Debug(_ context.Context, i ...interface{}) {
	zap.S().Debug(fmt.Sprint(i...))
}

func (l *Logger) Info(_ context.Context, i ...interface{}) {
	zap.S().Info(fmt.Sprint(i...))
}

func (l *Logger) Warn(_ context.Context, i ...interface{}) {
	zap.S().Warn(fmt.Sprint(i...))
}

func (l *Logger) Error(_ context.Context, i ...interface{}) {
	zap.S().Error(fmt.Sprint(i...))
}
