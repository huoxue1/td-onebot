package main

import (
	"flag"
	nested "github.com/Lyrics-you/sail-logrus-formatter/sailor"
	log "github.com/sirupsen/logrus"
	"td-onebot/conf"
	"td-onebot/lib"
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
	lib.Init()
}
