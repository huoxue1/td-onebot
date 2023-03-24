package lib

import (
	"context"
	"encoding/json"
	"github.com/botuniverse/go-libonebot"
	"github.com/gotd/td/tg"
	log "github.com/sirupsen/logrus"
	"strings"
	"td-onebot/models"
	"td-onebot/utils"
	"time"
)

func handleEvent(ob *Bot) {
	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		if _, ok := update.Message.(*tg.Message); ok {
			handleMessage(ob, update.Message.(*tg.Message), "private")

		} else {
			data, _ := json.Marshal(update)
			log.Infoln(string(data))
		}
		return nil
	})
	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewChannelMessage) error {
		if _, ok := update.Message.(*tg.Message); ok {
			handleMessage(ob, update.Message.(*tg.Message), "group")
		} else {
			data, _ := json.Marshal(update)
			log.Infoln(string(data))
		}

		return nil
	})

}

type MyMessageEvent struct {
	libonebot.MessageEvent
	UserID    string `json:"user_id"`
	ChannelID string `json:"channel_id"`
	SubType   string `json:"sub_type"`
}

func handleMessage(bot *Bot, msg *tg.Message, messageType string) {
	var messages libonebot.Message

	data, _ := json.Marshal(msg)
	log.Infoln(string(data))

	if msg.Entities != nil {
		offset := 0
		for _, entity := range msg.Entities {
			switch entity.(type) {
			case *tg.MessageEntityMention:
				mention := entity.(*tg.MessageEntityMention)
				name := strings.TrimPrefix(string([]byte(msg.Message)[mention.Offset-offset:mention.Offset-offset+mention.Length]), "@")
				msg.Message = string([]byte(msg.Message)[0:mention.Offset-offset]) + string([]byte(msg.Message)[mention.Offset-offset+mention.Length:])
				offset += mention.Length
				if name == bot.Self.Username {
					messages = append(messages, libonebot.MentionSegment(utils.ToString(bot.Self.ID)))
				} else {
					messages = append(messages, libonebot.MentionSegment("bot_"+name))
				}
			case *tg.MessageEntityMentionName:
				mention := entity.(*tg.MessageEntityMentionName)
				msg.Message = string([]byte(msg.Message)[0:mention.Offset-offset]) + string([]byte(msg.Message)[mention.Offset-offset+mention.Length:])
				offset += mention.Length
				messages = append(messages, libonebot.MentionSegment(utils.ToString(mention.UserID)))

			}

		}
	}

	if msg.Message != "" {
		messages = append(messages, libonebot.TextSegment(msg.Message))
	}

	if msg.FromID == nil {
		// 如果来源为空并且是私聊消息，自身是人，则说明这条消息是自己的其他设备发送的
		if messageType == "private" {
			if !bot.Self.Bot {
				msg.FromID = &tg.PeerUser{UserID: bot.Self.ID}
			} else {
				msg.FromID = &tg.PeerUser{
					UserID: msg.PeerID.(*tg.PeerUser).GetUserID(),
				}
			}

		} else { // 再群内的来源为空的消息就是匿名消息
			msg.FromID = &tg.PeerUser{UserID: 66666666}
		}
	}

	// 说明这条消息是频道消息
	if _, ok := msg.FromID.(*tg.PeerChannel); ok {
		msg.FromID = &tg.PeerUser{UserID: msg.FromID.(*tg.PeerChannel).GetChannelID()}
		messageType = "channel"
	}

	if msg.ReplyTo.ReplyToMsgID != 0 {
		if messageType == "group" {
			channelMessages := bot.getMsg(msg.GetPeerID().(*tg.PeerChannel).GetChannelID(), msg.ReplyTo.ReplyToMsgID).(*tg.MessagesChannelMessages)
			message := channelMessages.GetMessages()[0].(*tg.Message)
			if message.FromID != nil {
				switch message.FromID.(type) {
				case *tg.PeerUser:
					messages = append(messages, libonebot.ReplySegment((&models.MessageID{utils.ToString(msg.GetID()) + "_" + utils.ToString(msg.PeerID.(*tg.PeerChannel).GetChannelID())}).String(), utils.ToString(message.FromID.(*tg.PeerUser).GetUserID())))
				case *tg.PeerChannel:
					messages = append(messages, libonebot.ReplySegment((&models.MessageID{utils.ToString(msg.GetID()) + "_666666"}).String(), utils.ToString(message.FromID.(*tg.PeerUser).GetUserID())))
				default:
					log.Warningln("未知来源的回复，已忽略")
				}
			} else {
				messages = append(messages, libonebot.ReplySegment((&models.MessageID{utils.ToString(msg.GetID()) + "_666666"}).String(), "666666"))
			}

		} else if messageType == "private" {
			messagesMessages, ok := bot.getMsg(0, msg.ReplyTo.ReplyToMsgID).(*tg.MessagesMessages)
			if ok {
				message, ok1 := messagesMessages.GetMessages()[0].(*tg.Message)
				if ok1 {
					if message.FromID != nil {
						messages = append(messages, libonebot.ReplySegment((&models.MessageID{utils.ToString(msg.GetID())}).String(), utils.ToString(message.FromID.(*tg.PeerUser).GetUserID())))
					} else {
						messages = append(messages, libonebot.ReplySegment((&models.MessageID{utils.ToString(msg.GetID())}).String(), utils.ToString(bot.Self.GetID())))
					}
				}

			}

		}
	}

	if msg.Media != nil {
		switch msg.Media.(type) {

		case *tg.MessageMediaPhoto: // 图片消息
			photo := msg.Media.(*tg.MessageMediaPhoto).Photo.(*tg.Photo)
			fileId := (&models.FileId{
				Id:            photo.GetID(),
				AccessHash:    photo.GetAccessHash(),
				FileReference: photo.GetFileReference(),
				PeerId:        msg.PeerID,
				FromId:        msg.FromID,
				MsgId:         msg.GetID(),
				MessageType:   messageType,
			}).String()
			messages = append(messages, libonebot.ImageSegment(fileId))
		case *tg.MessageMediaDocument: // 文件消息
			doc := msg.Media.(*tg.MessageMediaDocument).Document.(*tg.Document)

			fileId := (&models.FileId{
				Id:            doc.GetID(),
				AccessHash:    doc.GetAccessHash(),
				FileReference: doc.GetFileReference(),
				PeerId:        msg.PeerID,
				FromId:        msg.FromID,
				MsgId:         msg.GetID(),
				MessageType:   messageType,
			}).String()
			messages = append(messages, libonebot.FileSegment(fileId))
		case *tg.MessageMediaGeo: // 位置分享
			geo := msg.Media.(*tg.MessageMediaGeo).Geo.(*tg.GeoPoint)
			messages = append(messages, libonebot.LocationSegment(geo.Lat, geo.Long, "", ""))
		}
	}

	if messageType == "channel" {
		e := libonebot.MakeChannelMessageEvent(time.Now(), (&models.MessageID{utils.ToString(msg.GetID())}).String(), messages, msg.Message, utils.ToString(msg.FromID.(*tg.PeerUser).GetUserID()), utils.ToString(msg.PeerID.(*tg.PeerChannel).GetChannelID()), "")
		e.Self = bot.Ob.Self
		pushEvent(bot, &e)
	} else if messageType == "group" {
		e := libonebot.MakeGroupMessageEvent(time.Now(), (&models.MessageID{utils.ToString(msg.GetID()) + "_" + utils.ToString(msg.PeerID.(*tg.PeerChannel).GetChannelID())}).String(), messages, msg.Message, utils.ToString(msg.PeerID.(*tg.PeerChannel).GetChannelID()), utils.ToString(msg.FromID.(*tg.PeerUser).GetUserID()))
		e.Self = bot.Ob.Self
		pushEvent(bot, &e)
	} else if messageType == "private" {
		e := libonebot.MakePrivateMessageEvent(time.Now(), (&models.MessageID{utils.ToString(msg.GetID())}).String(), messages, msg.Message, utils.ToString(msg.FromID.(*tg.PeerUser).GetUserID()))
		e.Self = bot.Ob.Self
		pushEvent(bot, &e)
	}
}

func pushEvent(ob *Bot, event libonebot.AnyEvent) {
	plugins.Range(func(key, value any) bool {
		value.(Service)(ob.ctx, ob, event)
		return true
	})
	log.Infoln(event)
	ob.Ob.Push(event)
}
