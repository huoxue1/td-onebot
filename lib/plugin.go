package lib

import (
	"context"
	"github.com/botuniverse/go-libonebot"
	log "github.com/sirupsen/logrus"
	"sync"
)

var (
	plugins sync.Map
)

type Service func(ctx context.Context, bot *Bot, event libonebot.AnyEvent)

func RegisterService(name string, plugin Service) {
	log.Infoln("已注册内置Service " + name)
	plugins.Store(name, plugin)
}
