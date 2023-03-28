package lib

import (
	"context"
	"encoding/json"
	"github.com/botuniverse/go-libonebot"
	"github.com/gotd/td/fileid"
	"github.com/gotd/td/tg"
	"github.com/huoxue1/td-onebot/utils"
	data2 "github.com/huoxue1/td-onebot/utils/const"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

func handleEvent(ob *Bot) {
	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		if _, ok := update.Message.(*tg.Message); ok {
			handlePrivateMsg(ob, update.Message.(*tg.Message))

		} else {
			data, _ := json.Marshal(update)
			log.Infoln(string(data))
		}
		return nil
	})
	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewChannelMessage) error {
		if _, ok := update.Message.(*tg.Message); ok {
			handleGroupMsg(ob, update.Message.(*tg.Message))
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

type option struct {
	userId    int64
	groupId   int64
	peerId    int64
	peerType  string
	messageId string
}

func handlePrivateMsg(bot *Bot, msg *tg.Message) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorln("handle private message error")
		}
	}()
	o := new(option)
	if msg.Out {
		o.userId = bot.Self.ID
		o.peerType = "private"
		o.peerId = msg.PeerID.(*tg.PeerUser).GetUserID()
	} else {
		o.userId = msg.PeerID.(*tg.PeerUser).GetUserID()
		o.peerType = "private"
		o.peerId = o.userId
	}
	o.messageId = (&utils.MessageID{utils.ToString(msg.GetID())}).String()
	log.Infof("%#v", o)
	handleMessage(bot, msg, o.peerType, o)
}

func handleGroupMsg(bot *Bot, msg *tg.Message) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorln("handle group message error")
		}

	}()
	o := new(option)
	if msg.Out {
		o.userId = bot.Self.ID
		o.peerType = "group"
		o.peerId = msg.PeerID.(*tg.PeerChannel).GetChannelID()
		o.groupId = o.peerId
	} else {
		if msg.FromID != nil {
			switch msg.FromID.(type) {
			case *tg.PeerUser:
				o.userId = msg.FromID.(*tg.PeerUser).GetUserID()
			case *tg.PeerChannel:
				o.userId = msg.FromID.(*tg.PeerChannel).GetChannelID()
			}

		} else {
			o.userId = data2.ManagerId
			o.peerType = "group"
		}

		o.peerId = msg.PeerID.(*tg.PeerChannel).GetChannelID()
		o.groupId = o.peerId
	}
	o.messageId = (&utils.MessageID{utils.ToString(msg.GetID()) + "_" + utils.ToString(o.peerId)}).String()
	log.Infof("%#v", o)
	handleMessage(bot, msg, o.peerType, o)

}

func handleMessage(bot *Bot, msg *tg.Message, messageType string, opt *option) {

	var messages libonebot.Message

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

	// 说明这条消息是频道消息
	if _, ok := msg.FromID.(*tg.PeerChannel); ok {
		msg.FromID = &tg.PeerUser{UserID: msg.FromID.(*tg.PeerChannel).GetChannelID()}
		messageType = "channel"
	}

	if msg.ReplyTo.ReplyToMsgID != 0 {
		if messageType == "group" {
			channelMessages := bot.getMsg(msg.GetPeerID().(*tg.PeerChannel).GetChannelID(), msg.ReplyTo.ReplyToMsgID).(*tg.MessagesChannelMessages)
			message, ok := channelMessages.GetMessages()[0].(*tg.Message)
			if !ok {
				return
			}
			if message.FromID != nil {
				switch message.FromID.(type) {
				case *tg.PeerUser:
					messages = append(messages, libonebot.ReplySegment((&utils.MessageID{utils.ToString(msg.GetID()) + "_" + utils.ToString(msg.PeerID.(*tg.PeerChannel).GetChannelID())}).String(), utils.ToString(message.FromID.(*tg.PeerUser).GetUserID())))
				case *tg.PeerChannel:
					messages = append(messages, libonebot.ReplySegment((&utils.MessageID{utils.ToString(msg.GetID()) + "_" + utils.ToString(data2.ManagerId)}).String(), utils.ToString(message.FromID.(*tg.PeerUser).GetUserID())))
				default:
					log.Warningln("未知来源的回复，已忽略")
				}
			} else {
				messages = append(messages, libonebot.ReplySegment((&utils.MessageID{utils.ToString(msg.GetID()) + "_" + utils.ToString(data2.ManagerId)}).String(), ""+utils.ToString(data2.ManagerId)))
			}

		} else if messageType == "private" {
			messagesMessages, ok := bot.getMsg(0, msg.ReplyTo.ReplyToMsgID).(*tg.MessagesMessages)
			if ok {
				message, ok1 := messagesMessages.GetMessages()[0].(*tg.Message)
				if ok1 {
					if message.FromID != nil {
						messages = append(messages, libonebot.ReplySegment((&utils.MessageID{utils.ToString(msg.GetID())}).String(), utils.ToString(message.FromID.(*tg.PeerUser).GetUserID())))
					} else {
						messages = append(messages, libonebot.ReplySegment((&utils.MessageID{utils.ToString(msg.GetID())}).String(), utils.ToString(bot.Self.GetID())))
					}
				}

			}

		}
	}

	if msg.Media != nil {
		switch msg.Media.(type) {

		case *tg.MessageMediaPhoto: // 图片消息
			photo := msg.Media.(*tg.MessageMediaPhoto).Photo.(*tg.Photo)
			fileId := (&utils.FileId{
				FileID:      fileid.FromPhoto(photo, 'x'),
				Type:        data2.FileIdTypeReceive,
				FileType:    data2.FileTypePhoto,
				MsgId:       msg.GetID(),
				MessageType: messageType,
				ChannelId:   opt.groupId,
			}).String()
			messages = append(messages, libonebot.ImageSegment(fileId))
		case *tg.MessageMediaDocument: // 文件消息
			doc := msg.Media.(*tg.MessageMediaDocument).Document.(*tg.Document)

			fileId := (&utils.FileId{
				FileID:      fileid.FromDocument(doc),
				Type:        data2.FileIdTypeReceive,
				FileType:    data2.FileTypeDocument,
				MsgId:       msg.GetID(),
				MessageType: messageType,
				ChannelId:   opt.groupId,
			}).String()
			messages = append(messages, libonebot.FileSegment(fileId))
		case *tg.MessageMediaGeo: // 位置分享
			geo := msg.Media.(*tg.MessageMediaGeo).Geo.(*tg.GeoPoint)
			messages = append(messages, libonebot.LocationSegment(geo.Lat, geo.Long, "", ""))
		}
	}

	if msg.Out {
		event := MakeSelfMessageEvent(messages, messages.ExtractText(), opt.messageId, opt.peerType, utils.ToString(opt.peerId), utils.ToString(bot.Self.ID))
		event.Self = bot.Ob.Self
		pushEvent(bot, &event)
	} else {
		if messageType == "channel" {
			e := libonebot.MakeChannelMessageEvent(time.Now(), (&utils.MessageID{utils.ToString(msg.GetID())}).String(), messages, msg.Message, utils.ToString(opt.groupId), utils.ToString(opt.groupId), "")
			e.Self = bot.Ob.Self
			pushEvent(bot, &e)
		} else if messageType == "group" {
			e := libonebot.MakeGroupMessageEvent(time.Now(), (&utils.MessageID{utils.ToString(msg.GetID()) + "_" + utils.ToString(opt.groupId)}).String(), messages, msg.Message, utils.ToString(opt.groupId), utils.ToString(opt.userId))
			e.Self = bot.Ob.Self
			pushEvent(bot, &e)
		} else if messageType == "private" {
			e := libonebot.MakePrivateMessageEvent(time.Now(), (&utils.MessageID{utils.ToString(msg.GetID())}).String(), messages, msg.Message, utils.ToString(opt.userId))
			e.Self = bot.Ob.Self
			pushEvent(bot, &e)
		}
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
