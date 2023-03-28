package main

import (
	"flag"
	nested "github.com/Lyrics-you/sail-logrus-formatter/sailor"
	"github.com/huoxue1/td-onebot/internal/cache"
	"github.com/huoxue1/td-onebot/internal/conf"
	"github.com/huoxue1/td-onebot/lib"
	log "github.com/sirupsen/logrus"
)

var (
	c string
)

func init() {
	flag.StringVar(&c, "config", "./config/", "the config path")
	flag.Parse()

	conf.InitConfig(c)

	log.SetFormatter(&nested.Formatter{
		FieldsOrder:           nil,
		TimeStampFormat:       "2006-01-02 15:04:05",
		CharStampFormat:       "",
		HideKeys:              false,
		Position:              true,
		Colors:                true,
		FieldsColors:          true,
		FieldsSpace:           true,
		ShowFullLevel:         false,
		LowerCaseLevel:        true,
		TrimMessages:          true,
		CallerFirst:           false,
		CustomCallerFormatter: nil,
	})
}

func main() {
	err := cache.InitCache(conf.GetConfig())
	if err != nil {
		log.Errorln("init cache error : " + err.Error())
		return
	}
	lib.Init()
}
