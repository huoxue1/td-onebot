package lib

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"github.com/botuniverse/go-libonebot"
	"github.com/google/uuid"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/huoxue1/td-onebot/internal/cache"
	"github.com/huoxue1/td-onebot/internal/conf"
	"github.com/huoxue1/td-onebot/utils"
	data2 "github.com/huoxue1/td-onebot/utils/const"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
)

type Bot struct {
	Self   *tg.User
	Ob     *libonebot.OneBot
	Client *telegram.Client
	ctx    context.Context
	config *conf.Config
	cache  cache.Client
}

func handleApi(bot *Bot) {
	mux := libonebot.NewActionMux()
	bot.Ob.Handle(mux)

	mux.HandleFunc(libonebot.ActionSendMessage, bot.SendMessage)

	mux.HandleFunc(libonebot.ActionUploadFile, bot.UploadFile)

	mux.HandleFunc(libonebot.ActionDeleteMessage, bot.DeleteMsg)

	mux.HandleFunc(libonebot.ActionGetFile, bot.GetFile())

	mux.HandleFunc(libonebot.ActionGetUserInfo, bot.GetUserInfo())

	mux.HandleFunc(libonebot.ActionGetGroupInfo, bot.GetGroupInfo())
	mux.HandleFunc(data2.ExtendActionEditMessage, bot.EditMessage)
	mux.HandleFunc(data2.ExtendActionGetDialogs, bot.GetDialogs())

}

func (b *Bot) GetGroupInfo() libonebot.HandlerFunc {
	return func(writer libonebot.ResponseWriter, request *libonebot.Request) {
		id, err := request.Params.GetString("group_id")
		if err != nil {
			writer.WriteFailed(libonebot.RetCodeBadParam, err)
			return
		}
		channel := b.getChannel(utils.ToInt64(id))
		writer.WriteData(map[string]any{
			"group_id":    utils.ToString(channel.GetID()),
			"group_name":  utils.ToString(channel.Username),
			"access_hash": utils.ToString(channel.AccessHash),
			"raw":         channel,
		})
	}
}

func (b *Bot) GetUserInfo() libonebot.HandlerFunc {
	return func(writer libonebot.ResponseWriter, request *libonebot.Request) {
		id, err := request.Params.GetString("user_id")
		if err != nil {
			writer.WriteFailed(libonebot.RetCodeBadParam, err)
			return
		}
		info, err := b.getUserInfo(utils.ToInt64(id))
		if err != nil {
			writer.WriteFailed(libonebot.RetCodeInternalHandlerError, err)
			return
		}
		writer.WriteData(map[string]any{
			"user_id":          utils.ToString(info.ID),
			"user_name":        info.Username,
			"is_bot":           info.GetBot(),
			"phone":            info.Phone,
			"status":           info.Status,
			"user_displayname": "",
			"user_remark":      "",
		})
	}
}

func (b *Bot) GetFile() libonebot.HandlerFunc {
	return func(writer libonebot.ResponseWriter, request *libonebot.Request) {
		id, err := request.Params.GetString("file_id")
		if err != nil {
			writer.WriteFailed(libonebot.RetCodeBadParam, err)
			return
		}
		fileId, err := utils.ParseFieldId(id)
		if err != nil {
			writer.WriteFailed(libonebot.RetCodeBadParam, err)
			return
		}
		dowType, err := request.Params.GetString("type")
		if err != nil {
			writer.WriteFailed(libonebot.RetCodeBadParam, err)
			return
		}
		thumbSize, _ := request.Params.GetString("thumb_size")
		builder, err := b.getFile(fileId, thumbSize)
		if err != nil {
			writer.WriteFailed(libonebot.RetCodeInternalHandlerError, err)
			return
		}
		switch dowType {
		case "path":
			if p := b.cache.Get(data2.CacheFile + id); p != "" {
				writer.WriteData(map[string]any{"name": filepath.Base(p), "path": p})
				return
			}
			_ = os.Mkdir(filepath.Join(b.config.Cache.CacheDir, "files"), 0644)
			u := uuid.New()
			abs, err := filepath.Abs(filepath.Join(b.config.Cache.CacheDir, "files", u.String()))
			if err != nil {
				writer.WriteFailed(libonebot.RetCodeInternalHandlerError, err)
				return
			}
			_, err = builder.ToPath(b.ctx, abs)
			if err != nil {
				writer.WriteFailed(libonebot.RetCodeInternalHandlerError, err)
				return
			}
			_ = b.cache.Set(data2.CacheFile+id, abs)
			writer.WriteData(map[string]any{"name": u, "path": abs})
		case "data":
			var buffer []byte
			buf := bytes.NewBuffer(buffer)
			_, err = builder.Stream(b.ctx, buf)
			if err != nil {
				writer.WriteFailed(libonebot.RetCodeInternalHandlerError, err)
				return
			}
			writer.WriteData(map[string]any{"name": "", "path": buf.Bytes()})
		default:
			writer.WriteFailed(libonebot.RetCodeUnsupportedParam, errors.New("unSupport the type "+dowType))
		}

	}
}

func (b *Bot) GetDialogs() libonebot.HandlerFunc {
	return func(writer libonebot.ResponseWriter, request *libonebot.Request) {
		limit, _ := request.Params.GetInt64("limit")
		if limit == 0 {
			limit = 100
		}
		dialogs, err := b.getDialogs(int(limit))
		if err != nil {
			writer.WriteFailed(libonebot.RetCodeInternalHandlerError, err)
			return
		}
		writer.WriteData(dialogs)
	}
}

func (b *Bot) EditMessage(resp libonebot.ResponseWriter, req *libonebot.Request) {
	id, err := req.Params.GetString("message_id")
	if err != nil {
		resp.WriteFailed(libonebot.RetCodeBadParam, err)
		return
	}
	messageId, err := utils.ParseMessageId(id)
	if err != nil {
		resp.WriteFailed(libonebot.RetCodeBadParam, err)
		return
	}
	msg, err := req.Params.GetString("message")
	if err != nil || msg == "" {
		resp.WriteFailed(libonebot.RetCodeBadParam, err)
		return
	}
	if len(messageId) > 1 {
		// TODO
	} else {
		err := b.editMessage(messageId[0], msg)
		if err != nil {
			resp.WriteFailed(libonebot.RetCodeInternalHandlerError, err)
			return
		}
	}
	resp.WriteOK()
}

func (b *Bot) DeleteMsg(resp libonebot.ResponseWriter, req *libonebot.Request) {
	id, err := req.Params.GetString("message_id")
	if err != nil {
		resp.WriteFailed(libonebot.RetCodeBadParam, err)
		return
	}
	messageId, err := utils.ParseMessageId(id)
	if err != nil {
		resp.WriteFailed(libonebot.RetCodeBadParam, err)
		return
	}
	if len(messageId) == 0 {
		resp.WriteFailed(libonebot.RetCodeBadParam, err)
		return
	}
	var ids []int
	var channeldId int64
	if strings.Contains(messageId[0], "_") {
		channeldId = utils.ToInt64(strings.Split(messageId[0], "_")[1])
		for _, s := range messageId {
			ids = append(ids, utils.ToInt(strings.Split(s, "_")[0]))
		}
	} else {
		for _, s := range messageId {
			ids = append(ids, utils.ToInt(s))
		}
	}
	err = b.deleteMsg(ids, channeldId)
	if err != nil {
		resp.WriteFailed(libonebot.RetCodeInternalHandlerError, err)
		return
	}
	resp.WriteOK()

}

func (b *Bot) deleteMsg(ids []int, channelId int64) error {
	if channelId != 0 {
		channel := b.getChannel(channelId)
		_, err := b.Client.API().ChannelsDeleteMessages(b.ctx, &tg.ChannelsDeleteMessagesRequest{
			Channel: &tg.InputChannel{
				ChannelID:  channel.ID,
				AccessHash: channel.AccessHash,
			},
			ID: ids,
		})
		if err != nil {
			return err
		}
	} else {
		_, err := b.Client.API().MessagesDeleteMessages(b.ctx, &tg.MessagesDeleteMessagesRequest{
			Revoke: true,
			ID:     ids,
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
		messageId, err := b.sendMessageCustom(detailType, utils.ToInt64(userId), 0, msg)
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
		messageId, err := b.sendMessageCustom(detailType, 0, utils.ToInt64(groupId), msg)
		if err != nil {
			resp.WriteFailed(libonebot.RetCodeInternalHandlerError, err)
			return
		}
		resp.WriteData(map[string]any{"message_id": messageId})
	default:
		resp.WriteFailed(libonebot.RetCodeUnsupportedParam, errors.New("the detailType unSupport"))
	}

}
