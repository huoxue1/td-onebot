package lib

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/botuniverse/go-libonebot"
	"github.com/gotd/td/bin"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"github.com/huoxue1/td-onebot/utils"
	data2 "github.com/huoxue1/td-onebot/utils/const"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

func (b *Bot) getFile(fileId *utils.FileId, thumbSize string) (*downloader.Builder, error) {
	if fileId.Type == data2.FileIdTypeUpload {
		return nil, errors.New("unSupport get the upload file")
	}
	dow := downloader.NewDownloader()
	location, b2 := fileId.AsInputFileLocation()
	if !b2 {
		return nil, errors.New("not AsInputFileLocation")
	}
	if ph, ok := location.(*tg.InputPhotoFileLocation); ok && thumbSize != "" {
		ph.ThumbSize = thumbSize
	}
	download := dow.Download(b.Client.API(), location)
	return download, nil
}

func (b *Bot) getUserInfo(userId int64) (*tg.User, error) {
	users, err := b.Client.API().UsersGetUsers(b.ctx, []tg.InputUserClass{&tg.InputUser{
		UserID: userId,
	}})
	if err != nil {
		return nil, err
	}
	user, ok := users[0].(*tg.User)
	if !ok {
		return nil, errors.New("the return type is not user")
	}
	return user, nil

}

func (b *Bot) getDialogs(limit int) (*tg.MessagesDialogs, error) {
	dialogs, err := b.Client.API().MessagesGetDialogs(b.ctx, &tg.MessagesGetDialogsRequest{
		Limit:         limit,
		ExcludePinned: true,
		OffsetPeer:    &tg.InputPeerSelf{},
	})
	if err != nil {
		return nil, err
	}
	switch dialogs.(type) {
	case *tg.MessagesDialogs:
		dia := dialogs.(*tg.MessagesDialogs)
		return dia, nil
	default:
		log.Errorln("unSupport dialogs type")
		return nil, errors.New("unSupport dialogs type")
	}
}

func (b *Bot) editMessage(msgId, msg string) error {
	req := &tg.MessagesEditMessageRequest{
		Message: msg,
	}
	if strings.Contains(msgId, "_") {
		req.ID = utils.ToInt(strings.Split(msgId, "_")[0])
		channel := b.getChannel(utils.ToInt64(strings.Split(msgId, "_")[1]))
		req.Peer = &tg.InputPeerChannel{
			ChannelID:  channel.GetID(),
			AccessHash: channel.AccessHash,
		}
	} else {
		req.ID = utils.ToInt(msgId)
		req.Peer = &tg.InputPeerSelf{}
	}
	_, err := b.Client.API().MessagesEditMessage(b.ctx, req)
	return err
}

func (b *Bot) sendMessageCustom(detailType string, userId int64, groupId int64, msg libonebot.Message) (string, error) {

	msgUtil := &utils.MsgUtil{Message: msg}
	var sender tg.InputPeerClass
	var text string
	var enties []tg.MessageEntityClass
	if detailType == "private" {
		sender = &tg.InputPeerUser{
			UserID: userId,
		}
	} else if detailType == "group" {
		sender = &tg.InputPeerChannel{
			ChannelID:  groupId,
			AccessHash: b.getChannel(groupId).AccessHash,
		}
	} else {
		return "", errors.New("the detail_type unSupport")
	}
	text = msgUtil.GetAllText()

	mentions := msgUtil.GetMentions()
	for _, mention := range mentions {
		info, err := b.getUserInfo(utils.ToInt64(mention))
		if err != nil {
			continue
		}
		text = fmt.Sprintf("@%s ", info.Username) + text
		enties = append(enties, &tg.InputMessageEntityMentionName{
			Offset: 0,
			Length: len(info.Username) + 1,
			UserID: &tg.InputUser{
				UserID:     info.ID,
				AccessHash: info.AccessHash,
			},
		})

	}
	if msgUtil.HasRichText() {
		result, err := b.ParseHtml(text)
		if err != nil {
			log.Errorln("parse html error : " + err.Error())
			return "", err
		}
		text = result.message
		for _, node := range result.nodes {
			enties = append(enties, node.Entities)
		}
	}
	if msgUtil.OnlyText() {
		replyId, _ := msgUtil.GetReplyId()
		t := &tg.MessagesSendMessageRequest{
			Peer:         sender,
			ReplyToMsgID: replyId,
			Message:      text,
			Entities:     []tg.MessageEntityClass{},
			RandomID:     time.Now().UnixMicro(),
		}

		updatesClass, err := b.Client.API().MessagesSendMessage(b.ctx, t)
		if err != nil {
			return "", err
		}
		return b.handleUpdatesToMessageId([]tg.UpdatesClass{updatesClass}).String(), nil
	}

	var (
		updates []tg.UpdatesClass
	)

	if msgUtil.HasFile() {
		files := msgUtil.GetFiles()
		var medias []tg.InputSingleMedia

		for _, f := range files {
			document := &tg.InputMediaUploadedDocument{
				File: f.InputFile,
			}
			messageMediaClass, err := b.Client.API().MessagesUploadMedia(b.ctx, &tg.MessagesUploadMediaRequest{
				Peer:  sender,
				Media: document,
			})
			if err != nil {
				log.Errorln(err.Error())
				continue
			}
			value, ok := messageMediaClass.(*tg.MessageMediaDocument).GetDocument()
			if !ok {
				continue
			}
			doc := value.(*tg.Document)

			medias = append(medias, tg.InputSingleMedia{
				RandomID: time.Now().UnixMicro() + rand.Int63n(1000000),
				Media: &tg.InputMediaDocument{
					ID: &tg.InputDocument{
						ID:            doc.ID,
						AccessHash:    doc.AccessHash,
						FileReference: doc.FileReference,
					},
				},
				Entities: enties,
				Message:  text,
			})
		}

		id, _ := msgUtil.GetReplyId()
		req := &tg.MessagesSendMultiMediaRequest{
			Flags:                  0,
			Silent:                 false,
			Background:             false,
			ClearDraft:             false,
			Noforwards:             false,
			UpdateStickersetsOrder: false,
			Peer:                   sender,
			ReplyToMsgID:           id,
			TopMsgID:               0,
			MultiMedia:             medias,
			ScheduleDate:           0,
			SendAs:                 sender,
		}
		updatesClass, err := b.Client.API().MessagesSendMultiMedia(b.ctx, req)
		if err != nil {
			return "", err
		}
		updates = append(updates, updatesClass)
	}

	if msgUtil.HasImage() {
		images := msgUtil.GetImages()
		var medias []tg.InputSingleMedia

		for _, image := range images {
			var pho *tg.Photo

			if image.Type == data2.FileIdTypeUpload {
				var photo tg.InputMediaClass
				photo = &tg.InputMediaUploadedPhoto{
					File: image.InputFile,
				}
				messageMediaClass, err := b.Client.API().MessagesUploadMedia(b.ctx, &tg.MessagesUploadMediaRequest{
					Peer:  sender,
					Media: photo,
				})
				if err != nil {
					continue
				}
				value, ok := messageMediaClass.(*tg.MessageMediaPhoto).GetPhoto()
				if !ok {
					continue
				}
				pho = value.(*tg.Photo)
			} else {

				pho = &tg.Photo{
					ID:            image.ID,
					AccessHash:    image.AccessHash,
					FileReference: image.FileReference,
				}
			}

			medias = append(medias, tg.InputSingleMedia{
				RandomID: time.Now().UnixMicro() + rand.Int63n(1000000),
				Media: &tg.InputMediaPhoto{
					ID: &tg.InputPhoto{
						ID:            pho.ID,
						AccessHash:    pho.AccessHash,
						FileReference: pho.FileReference,
					},
				},
				Message:  text,
				Entities: enties,
			})
		}
		id, _ := msgUtil.GetReplyId()
		req := &tg.MessagesSendMultiMediaRequest{
			Peer:         sender,
			ReplyToMsgID: id,
			TopMsgID:     0,
			MultiMedia:   medias,
			ScheduleDate: 0,
		}
		updatesClass, err := b.Client.API().MessagesSendMultiMedia(b.ctx, req)
		if err != nil {
			return "", err
		}
		updates = append(updates, updatesClass)
	}

	return b.handleUpdatesToMessageId(updates).String(), nil
}

func (b *Bot) handleUpdatesToMessageId(updates []tg.UpdatesClass) *utils.MessageID {
	var messageId utils.MessageID
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
					for _, u := range t.Updates {
						id, ok := u.(*tg.UpdateMessageID)
						if ok {
							messageId = append(messageId, utils.ToString(id.GetID())+"_"+utils.ToString(channel.GetID()))
						}
					}

				} else {
					for _, u := range t.Updates {
						id, ok := u.(*tg.UpdateMessageID)
						if ok {
							messageId = append(messageId, utils.ToString(id.GetID()))
						}
					}
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

func (b *Bot) uploadFile(data string, name string) (*utils.FileId, error) {
	newUploader := uploader.NewUploader(b.Client.API())
	uploadId := new(utils.FileId)
	var read io.Reader
	var fileClass tg.InputFileClass
	if strings.HasPrefix(data, "file:///") {
		file, err := os.Open(strings.TrimPrefix(data, "file:///"))
		if err != nil {
			return nil, err
		}
		defer func(file *os.File) {
			_ = file.Close()
		}(file)
		read = file
	} else if strings.HasPrefix(data, "http") || strings.HasPrefix(data, "https://") {
		response, err := http.Get(data)
		if err != nil {
			return nil, err
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(response.Body)
		read = response.Body
	} else if strings.HasPrefix(data, "base64://") {
		content, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, "base64://"))
		if err != nil {
			return nil, err
		}
		read = bytes.NewBuffer(content)
	} else {
		return nil, errors.New("fileType not support")
	}
	//md5 := utils.ReaderMd5(read)

	f, err := newUploader.FromReader(b.ctx, name, read)
	if err != nil {
		return nil, err
	}
	fileClass = f

	uploadId.InputFile = fileClass
	uploadId.FileName = name
	uploadId.Type = data2.FileIdTypeUpload
	return uploadId, nil
}

var (
	excludeType = []string{
		"a", "b", "i", "at", "tel", "code", "pre",
	}
)

type Node struct {
	Text     string
	Type     string
	State    map[string]string
	Entities tg.MessageEntityClass
}

type Result struct {
	message string
	nodes   []*Node
}

func (b *Bot) ParseHtml(data string) (*Result, error) {
	node, err := html.ParseFragmentWithOptions(strings.NewReader(data), &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	}, html.ParseOptionEnableScripting(false))
	if err != nil {
		return nil, err
	}
	result := new(Result)
	for _, n := range node {
		if utils.In(n.Data, excludeType) || n.Type == html.TextNode {
			if n.Type == html.TextNode {
				result.message += n.Data
				continue
			}
			n2 := new(Node)
			attribute := make(map[string]string, len(n.Attr))
			for _, att := range n.Attr {
				attribute[att.Key] = att.Val
			}
			n2.State = attribute
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				n2.Text = n.FirstChild.Data
			}
			switch n.Data {
			case "at":
				id, ok := attribute["user_id"]
				if !ok {
					continue
				}
				info, err := b.getUserInfo(utils.ToInt64(id))
				if err != nil {
					continue
				}
				if n2.Text == "" {
					n2.Text = "@" + info.Username
				}
				n2.Entities = &tg.InputMessageEntityMentionName{
					Offset: len(result.message),
					Length: len(n2.Text),
					UserID: info.AsInput(),
				}
				result.message += n2.Text + " "
				result.nodes = append(result.nodes, n2)
			case "a":
				href, ok := attribute["href"]
				if !ok {
					continue
				}
				if n2.Text == "" {
					n2.Text = href
				}
				n2.Entities = &tg.MessageEntityTextURL{
					Offset: len(result.message),
					Length: len(n2.Text),
					URL:    href,
				}
				result.message += n2.Text
				result.nodes = append(result.nodes, n2)
			case "b":
				if n2.Text == "" {
					continue
				}
				n2.Entities = &tg.MessageEntityBold{
					Offset: len(result.message),
					Length: len(n2.Text),
				}
				result.message += n2.Text
				result.nodes = append(result.nodes, n2)
			case "i":
				if n2.Text == "" {
					continue
				}
				n2.Entities = &tg.MessageEntityItalic{
					Offset: len(result.message),
					Length: len(n2.Text),
				}
				result.message += n2.Text
				result.nodes = append(result.nodes, n2)
			case "u":
				if n2.Text == "" {
					continue
				}
				n2.Entities = &tg.MessageEntityUnderline{
					Offset: len(result.message),
					Length: len(n2.Text),
				}
				result.message += n2.Text
				result.nodes = append(result.nodes, n2)
			case "tel":
				if n2.Text == "" {
					continue
				}
				n2.Entities = &tg.MessageEntityPhone{
					Offset: len(result.message),
					Length: len(n2.Text),
				}
				result.message += n2.Text
				result.nodes = append(result.nodes, n2)
			case "code":
				if n2.Text == "" {
					continue
				}
				n2.Entities = &tg.MessageEntityCode{
					Offset: len(result.message),
					Length: len(n2.Text),
				}
				result.message += n2.Text
				result.nodes = append(result.nodes, n2)
			case "pre":
				if n2.Text == "" {
					continue
				}
				language := attribute["language"]

				n2.Entities = &tg.MessageEntityPre{
					Offset:   len(result.message),
					Length:   len(n2.Text),
					Language: language,
				}
				result.message += n2.Text
				result.nodes = append(result.nodes, n2)

			default:
				log.Errorln("unSupport node " + n.Data)
			}

		}
	}
	return result, nil
}
