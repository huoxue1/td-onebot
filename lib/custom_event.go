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

func MakeSelfMessageEvent(message libonebot.Message, altMessage string, messageId string, peerType, peerId string, selfId string) SelfMessageEvent {
	return SelfMessageEvent{
		MessageEvent: libonebot.MakeMessageEvent(time.Now(), "self_message", messageId, message, altMessage),
		PeerId:       peerId,
		PeerType:     peerType,
		UserId:       selfId,
	}

}

type SelfMessageEvent struct {
	libonebot.MessageEvent
	PeerId   string `json:"peer_id"`
	PeerType string `json:"peer_type"`
	UserId   string `json:"user_id"`
}
