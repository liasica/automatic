// Copyright (C) automatic. 2026-present.
//
// Created at 2026-02-25, by liasica

package core

import (
	"os"

	"sigs.k8s.io/yaml"

	"automatic/util"
)

type Config struct {
	Redis   *Redis
	Lark    *Lark
	Openwrt *Openwrt
	Users   []*User
}

type Redis struct {
	Addr string
	DB   int
}

type Lark struct {
	AppId     string
	AppSecret string
}

type Openwrt struct {
	Url string
}

type User struct {
	Id           string   // 飞书用户ID
	MacAddresses []string // 设备MAC地址
	CheckIn      CheckIn  // 上班打卡时间随机范围（分钟，向前取值，例如设备在线时间是9:00，CheckIn Range是{From: 0, To: 30}，则打卡时间会在8:30-9:00之间随机取值）
	CheckOut     CheckOut // 下班打卡时间随机范围（分钟，向后取值，例如设备在线时间是18:00，CheckOut Range是{From: 0, To: 30}，则打卡时间会在18:00-18:30之间随机取值）
}

type CheckIn struct {
	Latest string // 上班打卡时间最晚时间，例如"09:00"
	From   int
	To     int

	LatestTime HourMinute
}

type CheckOut struct {
	Earliest string // 下班打卡时间最早时间，例如"18:00"
	From     int
	To       int

	EarliestTime HourMinute
}

type HourMinute struct {
	Hour   int
	Minute int
}

func NewConfig(configPath string) (cfg *Config, err error) {
	var data []byte
	data, err = os.ReadFile(configPath)
	if err != nil {
		return
	}

	cfg = &Config{}
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return
	}

	for _, user := range cfg.Users {
		var hour, minute int
		hour, minute, err = util.ParseHourMinute(user.CheckIn.Latest)
		if err != nil {
			return
		}

		user.CheckIn.LatestTime = HourMinute{
			Hour:   hour,
			Minute: minute,
		}

		hour, minute, err = util.ParseHourMinute(user.CheckOut.Earliest)
		if err != nil {
			return
		}

		user.CheckOut.EarliestTime = HourMinute{
			Hour:   hour,
			Minute: minute,
		}
	}
	return
}
