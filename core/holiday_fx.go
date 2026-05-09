// Copyright (C) automatic. 2026-present.
//
// Created at 2026-05-09, by liasica

package core

import (
	"context"

	"go.uber.org/fx"
)

// HolidayModule 节假日模块：构造 manager + 在 fx Lifecycle 上挂启停钩子
// 同时把 manager 安装为包级单例，使包级 GetDayType 自动走真实数据
var HolidayModule = fx.Module("holiday",
	fx.Provide(NewHolidayManager),
	fx.Invoke(registerHolidayLifecycle),
)

// registerHolidayLifecycle 把 manager 的 Start/Stop 接到 fx Lifecycle 上
// 同时调用 SetHolidayManager 安装包级单例
func registerHolidayLifecycle(lc fx.Lifecycle, m HolidayManager) {
	hm, ok := m.(*holidayManager)
	if !ok {
		// 测试或自定义实现：仅安装单例，不挂启停
		SetHolidayManager(m)
		return
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := hm.Start(ctx); err != nil {
				return err
			}
			SetHolidayManager(m)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			SetHolidayManager(nil)
			return hm.Stop(ctx)
		},
	})
}
