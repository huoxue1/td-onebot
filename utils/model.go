package utils

import (
	"encoding/base64"
	"encoding/json"
	"github.com/gotd/td/fileid"
	"github.com/gotd/td/tg"
	log "github.com/sirupsen/logrus"
)

type FileId struct {
	UploadId
	fileid.FileID
	Type        string `json:"type,omitempty"`
	FileType    string `json:"file_type,omitempty"`
	MsgId       int    `json:"msg_id,omitempty"`
	MessageType string `json:"message_type,omitempty"`
	ChannelId   int64  `json:"channel_id,omitempty"`
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
	data, err := EncodeObject(u)
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
	err = DecodeObject(content, f)
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
