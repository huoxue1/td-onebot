package models

import (
	"encoding/base64"
	"encoding/json"
	"github.com/gotd/td/tg"
	log "github.com/sirupsen/logrus"
	"td-onebot/utils"
)

type FileId struct {
	Id            int64        `json:"id"`
	AccessHash    int64        `json:"access_hash"`
	FileReference []byte       `json:"file_reference"`
	PeerId        tg.PeerClass `json:"peer_id"`
	FromId        tg.PeerClass `json:"from_id"`
	MsgId         int          `json:"msg_id"`
	MessageType   string       `json:"message_type"`
}

func (f *FileId) String() string {
	data, _ := json.Marshal(f)
	return base64.StdEncoding.EncodeToString(data)
}

func ParseFieldId(data string) (*FileId, error) {
	f := new(FileId)
	content, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(content, f)
	if err != nil {
		return nil, err
	}
	return f, nil
}

type UploadId struct {
	FileName  string            `json:"file_name"`
	InputFile tg.InputFileClass `json:"input_file"`
}

func (u *UploadId) String() string {
	data, err := utils.EncodeObject(u)
	if err != nil {
		log.Errorln(err.Error())
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

func ParseUploadId(data string) (*UploadId, error) {
	f := new(UploadId)
	content, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	err = utils.DecodeObject(content, f)
	if err != nil {
		return nil, err
	}
	return f, nil
}

type MessageID []string

func (m *MessageID) String() string {
	data, _ := json.Marshal(m)
	return base64.StdEncoding.EncodeToString(data)
}

func ParseMessageId(data string) (MessageID, error) {
	var m MessageID
	content, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(content, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
