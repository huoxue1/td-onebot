package lib

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/botuniverse/go-libonebot"
	"github.com/gotd/td/bin"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strings"
	"td-onebot/models"
	"td-onebot/utils"
)

func (b *Bot) sendMessage(detailType string, userId int64, groupId int64, msg libonebot.Message) (string, error) {
	var builder message.Builder
	if detailType == "private" {
		builder = message.NewSender(b.Client.API()).To(&tg.InputPeerUser{
			UserID: userId,
		}).Builder
	} else if detailType == "group" {
		builder = message.NewSender(b.Client.API()).To(&tg.InputPeerChannel{
			ChannelID:  groupId,
			AccessHash: b.getChannel(groupId).AccessHash,
		}).Builder
	} else {
		return "", errors.New("the detail_type unSupport")
	}

	//var messageId models.MessageID

	build := &builder
	replyId := utils.GetReplyId(msg)
	if replyId != "" {
		messageId, err := models.ParseMessageId(replyId)
		if err != nil {
			return "", fmt.Errorf("parse message_id:%w", err)
		}
		if detailType == "private" {
			build = builder.Reply(utils.ToInt(messageId[0]))
		} else {
			build = builder.Reply(utils.ToInt(strings.Split(messageId[0], "_")[0]))
		}

	}

	var updates []tg.UpdatesClass

	//var messageId models.MessageID
	for _, segment := range msg {
		if segment.Type == "image" {
			fileId, err := segment.Data.GetString("file_id")
			if err != nil {
				log.Errorln("已忽略的消息段，" + err.Error())
				continue
			}
			uploadId, err := models.ParseUploadId(fileId)
			if err != nil {
				log.Errorln("已忽略的消息段，" + err.Error())
				continue
			}
			photo, err := build.UploadedPhoto(b.ctx, uploadId.InputFile)
			if err != nil {
				log.Errorln("已忽略的消息段，" + err.Error())
				continue
			}
			updates = append(updates, photo)
			log.Infoln(photo)
		} else if segment.Type == "file" {
			fileId, err := segment.Data.GetString("file_id")
			if err != nil {
				log.Errorln("已忽略的消息段，" + err.Error())
				continue
			}
			uploadId, err := models.ParseUploadId(fileId)
			if err != nil {
				log.Errorln("已忽略的消息段，" + err.Error())
				continue
			}
			photo, err := build.Media(b.ctx, message.UploadedDocument(uploadId.InputFile))
			if err != nil {
				log.Errorln("已忽略的消息段，" + err.Error())
				continue
			}
			updates = append(updates, photo)

			log.Infof("%T", photo)
		}
	}
	text, err := build.Text(b.ctx, msg.ExtractText())
	if err != nil {
		return "", err
	}
	updates = append(updates, text)

	return b.handleUpdatesToMessageId(updates).String(), nil
}

func (b *Bot) handleUpdatesToMessageId(updates []tg.UpdatesClass) *models.MessageID {
	var messageId models.MessageID
	for _, update := range updates {
		switch update.(type) {
		case *tg.Updates:
			{
				t := update.(*tg.Updates)
				if t.Chats != nil {
					channel, ok := t.GetChats()[0].(*tg.Channel)
					if !ok {
						continue
					}
					id, ok := t.Updates[0].(*tg.UpdateMessageID)
					if !ok {
						continue
					}
					messageId = append(messageId, utils.ToString(id.GetID())+"_"+utils.ToString(channel.GetID()))

				} else {
					id, ok := t.Updates[0].(*tg.UpdateMessageID)
					if !ok {
						continue
					}
					messageId = append(messageId, utils.ToString(id.GetID()))
				}
			}
		case *tg.UpdateShortMessage:
			{
				t := update.(*tg.UpdateShortMessage)
				messageId = append(messageId, utils.ToString(t.GetID()))
			}

		}

	}
	return &messageId
}

func (b *Bot) getPhoto(id, accessHash int64, fileReference []byte) ([]byte, error) {
	file, err := b.Client.API().UploadGetFile(b.ctx, &tg.UploadGetFileRequest{
		Flags: bin.Fields(0),
		Location: &tg.InputPhotoFileLocation{
			ID:            id,
			AccessHash:    accessHash,
			FileReference: fileReference,
			ThumbSize:     "",
		},
		Limit: 1024 * 1024,
	})
	if err != nil {
		return nil, err
	}
	return file.(*tg.UploadFile).GetBytes(), nil
}

func (b *Bot) getDocument(id, accessHash int64, fileReference []byte) ([]byte, error) {
	file, err := b.Client.API().UploadGetFile(b.ctx, &tg.UploadGetFileRequest{
		Flags: bin.Fields(0),
		Location: &tg.InputDocumentFileLocation{
			ID:            id,
			AccessHash:    accessHash,
			FileReference: fileReference,
			ThumbSize:     "",
		},
		Limit: 1024 * 1024,
	})
	if err != nil {
		return nil, err
	}
	return file.(*tg.UploadFile).GetBytes(), nil
}

func (b *Bot) getMsg(channelID int64, msgID int) tg.MessagesMessagesClass {
	if channelID != 0 {
		messages, err := b.Client.API().ChannelsGetMessages(b.ctx, &tg.ChannelsGetMessagesRequest{
			Channel: &tg.InputChannel{
				ChannelID:  channelID,
				AccessHash: b.getChannel(channelID).AccessHash,
			},
			ID: []tg.InputMessageClass{&tg.InputMessageID{
				ID: msgID,
			}},
		})
		if err != nil {
			return nil
		}
		return messages

	} else {
		messages, err := b.Client.API().MessagesGetMessages(b.ctx, []tg.InputMessageClass{&tg.InputMessageID{
			ID: msgID,
		}})
		if err != nil {
			return nil
		}
		return messages
	}
}

func (b *Bot) getChannel(channelID int64) *tg.Channel {
	resolvedPeer, err := b.Client.API().ChannelsGetChannels(b.ctx, []tg.InputChannelClass{&tg.InputChannel{ChannelID: channelID}})
	if err != nil {
		log.Errorln(err.Error())
		return nil
	}

	channel := resolvedPeer.(*tg.MessagesChats).Chats[0].(*tg.Channel)
	return channel
}

func (b *Bot) uploadFile(data string, name string) (*models.UploadId, error) {
	newUploader := uploader.NewUploader(b.Client.API())
	uploadId := new(models.UploadId)
	if strings.HasPrefix(data, "file:///") {
		file, err := os.Open(strings.TrimPrefix(data, "file:///"))
		if err != nil {
			return nil, err
		}
		uploadId.FileName = file.Name()
		fromFile, err := newUploader.FromReader(b.ctx, name, file)
		if err != nil {
			return nil, err
		}
		uploadId.InputFile = fromFile
		return uploadId, nil
	} else if strings.HasPrefix(data, "http") || strings.HasPrefix(data, "https://") {
		response, err := http.Get(data)
		if err != nil {
			return nil, err
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(response.Body)
		fromURL, err := newUploader.FromReader(b.ctx, name, response.Body)
		if err != nil {
			return nil, err
		}
		uploadId.FileName = name
		uploadId.InputFile = fromURL
		return uploadId, nil
	} else if strings.HasPrefix(data, "base64://") {
		content, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, "base64://"))
		if err != nil {
			return nil, err
		}
		fromBytes, err := newUploader.FromBytes(b.ctx, name, content)
		if err != nil {
			return nil, err
		}
		uploadId.FileName = name
		uploadId.InputFile = fromBytes

	}
	return nil, errors.New("file_id not support")
}
