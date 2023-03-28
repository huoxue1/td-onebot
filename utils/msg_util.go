package utils

import (
	"github.com/botuniverse/go-libonebot"
	data "github.com/huoxue1/td-onebot/utils/const"
	log "github.com/sirupsen/logrus"
	"strings"
)

type MsgUtil struct {
	libonebot.Message
}

func (m *MsgUtil) GetAllText() (result string) {
	for _, segment := range m.Message {
		if segment.Type == libonebot.SegTypeText || segment.Type == data.SegmentRichText {
			text, err := segment.Data.GetString("text")
			if err != nil {
				continue
			}
			result += text
		}
	}
	return
}

func (m *MsgUtil) OnlyText() bool {

	exludedType := []string{libonebot.SegTypeMention, libonebot.SegTypeMentionAll, data.SegmentRichText}

	hasText, onlyText := false, true
	for _, msg := range m.Message {
		if msg.Type != libonebot.SegTypeText {
			if !In(msg.Type, exludedType) {
				onlyText = false
			}

		} else {
			hasText = true
		}
	}
	return onlyText && hasText
}

func (m *MsgUtil) HasText() bool {
	return m.hasSegment(libonebot.SegTypeText)
}

func (m *MsgUtil) HasRichText() bool {
	return m.hasSegment(data.SegmentRichText)
}

func (m *MsgUtil) hasSegment(segmentType string) bool {
	for _, segment := range m.Message {
		if segment.Type == segmentType {
			return true
		}
	}
	return false
}

func (m *MsgUtil) HasMention() bool {
	return m.hasSegment(libonebot.SegTypeMention)
}

func (m *MsgUtil) GetMentions() (result []string) {
	for _, segment := range m.Message {
		if segment.Type == libonebot.SegTypeMention {
			userId, err := segment.Data.GetString("user_id")
			if err != nil {
				log.Errorln("已忽略的消息段：" + err.Error())
				continue
			}
			result = append(result, userId)
		}
	}
	return
}

func (m *MsgUtil) HasImage() bool {
	return m.hasSegment(libonebot.SegTypeImage)
}

func (m *MsgUtil) HasFile() bool {
	return m.hasSegment(libonebot.SegTypeFile)
}

func (m *MsgUtil) GetImages() []*FileId {
	return m.getFiles(libonebot.SegTypeImage)
}

func (m *MsgUtil) GetFiles() []*FileId {
	return m.getFiles(libonebot.SegTypeFile)
}

func (m *MsgUtil) getFiles(segmentType string) []*FileId {

	var fileIds []*FileId
	for _, segment := range m.Message {
		if segment.Type == segmentType {
			fileId, err := segment.Data.GetString("file_id")
			if err != nil {
				log.Errorln("已忽略的消息段，" + err.Error())
				continue
			}
			uploadId, err := ParseFieldId(fileId)
			if err != nil {
				log.Errorln("已忽略的消息段，" + err.Error())
				continue
			}
			fileIds = append(fileIds, uploadId)
		}
	}
	return fileIds
}

func (m *MsgUtil) GetReplyId() (int, int64) {
	for _, segment := range m.Message {
		if segment.Type == "reply" {
			messageId, _ := segment.Data.GetString("message_id")
			id, err := ParseMessageId(messageId)
			if err != nil {
				return 0, 0
			}
			if strings.Contains(id[0], "_") {
				return ToInt(strings.Split(id[0], "_")[0]), ToInt64(strings.Split(id[0], "_")[1])
			} else {
				return ToInt(id[0]), 0
			}
		}
	}
	return 0, 0
}
