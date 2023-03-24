package utils

import "github.com/botuniverse/go-libonebot"

func GetReplyId(msg libonebot.Message) string {
	for _, segment := range msg {
		if segment.Type == "reply" {
			messageId, _ := segment.Data.GetString("message_id")
			return messageId
		}
	}
	return ""
}
