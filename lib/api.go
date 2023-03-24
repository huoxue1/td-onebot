package lib

import (
	"context"
	"encoding/base64"
	"errors"
	"github.com/botuniverse/go-libonebot"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	log "github.com/sirupsen/logrus"
	"strings"
	"td-onebot/models"
	"td-onebot/utils"
)

type Bot struct {
	Self   *tg.User
	Ob     *libonebot.OneBot
	Client *telegram.Client
	ctx    context.Context
	config map[string]any
}

func handleApi(bot *Bot) {
	mux := libonebot.NewActionMux()
	bot.Ob.Handle(mux)

	mux.HandleFunc(libonebot.ActionSendMessage, bot.SendMessage)

	mux.HandleFunc(libonebot.ActionUploadFile, bot.UploadFile)

	mux.HandleFunc(libonebot.ActionDeleteMessage, bot.DeleteMsg)

	mux.HandleFunc(libonebot.ActionGetSupportedActions, bot.GetSupportActions)

}

func (b *Bot) GetSupportActions(writer libonebot.ResponseWriter, req *libonebot.Request) {
	writer.WriteData([]string{
		libonebot.ActionSendMessage,
		libonebot.ActionUploadFile,
		libonebot.ActionDeleteMessage,
		libonebot.ActionGetSupportedActions,
	})

}

func (b *Bot) DeleteMsg(resp libonebot.ResponseWriter, req *libonebot.Request) {
	id, err := req.Params.GetString("message_id")
	if err != nil {
		resp.WriteFailed(libonebot.RetCodeBadParam, err)
		return
	}
	messageId, err := models.ParseMessageId(id)
	if err != nil {
		resp.WriteFailed(libonebot.RetCodeBadParam, err)
		return
	}
	var err1 error
	for _, s := range messageId {
		err1 = b.deleteMsg(s)
	}
	if err1 != nil {
		resp.WriteFailed(libonebot.RetCodeInternalHandlerError, err1)
		return
	}
	resp.WriteOK()

}

func (b *Bot) deleteMsg(id string) error {
	if strings.Contains(id, "_") {
		channel := b.getChannel(utils.ToInt64(strings.Split(id, "_")[1]))
		_, err := b.Client.API().ChannelsDeleteMessages(b.ctx, &tg.ChannelsDeleteMessagesRequest{
			Channel: &tg.InputChannel{
				ChannelID:  channel.ID,
				AccessHash: channel.AccessHash,
			},
			ID: []int{utils.ToInt(strings.Split(id, "_")[0])},
		})
		if err != nil {
			return err
		}
	} else {
		_, err := b.Client.API().MessagesDeleteMessages(b.ctx, &tg.MessagesDeleteMessagesRequest{
			Revoke: true,
			ID:     []int{utils.ToInt(id)},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Bot) UploadFile(resp libonebot.ResponseWriter, req *libonebot.Request) {
	fileType, err := req.Params.GetString("type")
	if err != nil {
		resp.WriteFailed(libonebot.RetCodeBadParam, err)
		return
	}
	name, err := req.Params.GetString("name")
	if err != nil {
		resp.WriteFailed(libonebot.RetCodeBadParam, err)
		return
	}
	var result string
	switch fileType {
	case "url":
		url, err := req.Params.GetString("url")
		if err != nil {
			resp.WriteFailed(libonebot.RetCodeBadParam, err)
			return
		}
		result = url
	case "path":
		path, err := req.Params.GetString("path")
		if err != nil {
			resp.WriteFailed(libonebot.RetCodeBadParam, err)
			return
		}
		result = "file:///" + path
	case "data":
		data, err := req.Params.GetBytes("data")
		if err != nil {
			resp.WriteFailed(libonebot.RetCodeBadParam, err)
			return
		}
		result = "base64://" + base64.StdEncoding.EncodeToString(data)

	default:
		resp.WriteFailed(libonebot.RetCodeUnsupportedParam, errors.New("the file type not support"))
		return
	}
	uploadId, err := b.uploadFile(result, name)
	if err != nil {
		resp.WriteFailed(libonebot.RetCodeInternalHandlerError, err)
		return
	}

	uploadId.FileName = name
	resp.WriteData(map[string]string{"file_id": uploadId.String()})
}

func (b *Bot) SendMessage(resp libonebot.ResponseWriter, req *libonebot.Request) {
	log.Infoln(req.Params)
	detailType, err := req.Params.GetString("detail_type")
	if err != nil {
		resp.WriteFailed(libonebot.RetCodeBadParam, err)
		return
	}
	msg, err := req.Params.GetMessage("message")
	if err != nil {
		resp.WriteFailed(libonebot.RetCodeBadParam, err)
		return
	}
	switch detailType {
	case "private":
		userId, err := req.Params.GetString("user_id")
		if err != nil {
			resp.WriteFailed(libonebot.RetCodeBadParam, err)
			return
		}
		messageId, err := b.sendMessage(detailType, utils.ToInt64(userId), 0, msg)
		if err != nil {
			resp.WriteFailed(libonebot.RetCodeInternalHandlerError, err)
			return
		}
		resp.WriteData(map[string]any{"message_id": messageId})
	case "group":
		groupId, err := req.Params.GetString("group_id")
		if err != nil {
			resp.WriteFailed(libonebot.RetCodeBadParam, err)
			return
		}
		messageId, err := b.sendMessage(detailType, 0, utils.ToInt64(groupId), msg)
		if err != nil {
			resp.WriteFailed(libonebot.RetCodeInternalHandlerError, err)
			return
		}
		resp.WriteData(map[string]any{"message_id": messageId})
	default:
		resp.WriteFailed(libonebot.RetCodeUnsupportedParam, errors.New("the detailType unSupport"))
	}

}
