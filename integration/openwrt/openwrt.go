// Copyright (C) automatic. 2026-present.
//
// Created at 2026-02-25, by liasica

package openwrt

import (
	"sync"
	"time"

	"resty.dev/v3"

	"automatic/core"
)

type DeviceFlags string

const (
	FlagsOffline DeviceFlags = "0x0"
	FlagsOnline  DeviceFlags = "0x2"
)

// EventType 设备事件类型
type EventType int

const (
	EventOnline  EventType = iota // 设备上线
	EventOffline                  // 设备离线
)

// DeviceEvent 设备上下线事件
type DeviceEvent struct {
	Type   EventType
	Device *Device
}

// EventHandler 事件处理函数
type EventHandler func(event DeviceEvent)

type Device struct {
	Ip    string      `json:"ip"`
	Mac   string      `json:"mac"`
	Flags DeviceFlags `json:"flags"` // 0x2 在线，0x0 或未获取到离线
}

type OpenWrt struct {
	client   *resty.Client
	mu       sync.RWMutex
	known    map[string]*Device // mac -> device，当前已知在线设备
	handlers []EventHandler
	stopCh   chan struct{}
}

func New(cfg *core.Config) *OpenWrt {
	return &OpenWrt{
		client: resty.New().SetBaseURL(cfg.Openwrt.Url),
		known:  make(map[string]*Device),
		stopCh: make(chan struct{}),
	}
}

// AddHandler 注册设备事件监听器，支持多个
func (o *OpenWrt) AddHandler(h EventHandler) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.handlers = append(o.handlers, h)
}

// emit 触发事件，通知所有监听器
func (o *OpenWrt) emit(event DeviceEvent) {
	o.mu.RLock()
	handlers := make([]EventHandler, len(o.handlers))
	copy(handlers, o.handlers)
	o.mu.RUnlock()

	for _, h := range handlers {
		h(event)
	}
}

// GetDevices 获取设备列表
func (o *OpenWrt) GetDevices() (devices []*Device, err error) {
	_, err = o.client.R().
		SetResult(&devices).
		Get("devices")
	return
}

// poll 拉取一次设备列表并与已知状态对比，触发变动事件
func (o *OpenWrt) poll() {
	devices, err := o.GetDevices()
	if err != nil {
		return
	}

	// 构建本次在线设备 map
	current := make(map[string]*Device, len(devices))
	for _, d := range devices {
		if d.Flags == FlagsOnline {
			current[d.Mac] = d
		}
	}

	o.mu.Lock()
	// 检测新上线设备
	for mac, d := range current {
		if _, existed := o.known[mac]; !existed {
			o.mu.Unlock()
			o.emit(DeviceEvent{Type: EventOnline, Device: d})
			o.mu.Lock()
		}
	}

	// 检测新离线设备
	for mac, d := range o.known {
		if _, stillOnline := current[mac]; !stillOnline {
			o.mu.Unlock()
			o.emit(DeviceEvent{Type: EventOffline, Device: d})
			o.mu.Lock()
		}
	}

	o.known = current
	o.mu.Unlock()
}

// Start 启动定时轮询，interval 为轮询间隔（建议 1*time.Minute）
func (o *OpenWrt) Start(interval time.Duration) {
	go func() {
		// 立即执行一次
		o.poll()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				o.poll()
			case <-o.stopCh:
				return
			}
		}
	}()
}

// Stop 停止定时轮询
func (o *OpenWrt) Stop() {
	close(o.stopCh)
}
