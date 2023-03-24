package lib

import (
	"github.com/botuniverse/go-libonebot"
	"time"
)

type MetaConnectEvent struct {
	libonebot.MetaEvent
	Version version `json:"version"`
}

type version struct {
	Impl          string `json:"impl"`
	Version       string `json:"version"`
	OnebotVersion string `json:"onebot_version"`
}

func MakeMetaConnectEvent(impl, v, onebotVersion string) *MetaConnectEvent {
	return &MetaConnectEvent{
		MetaEvent: libonebot.MakeMetaEvent(time.Now(), "connect"),
		Version:   version{Impl: impl, Version: v, OnebotVersion: onebotVersion},
	}
}
