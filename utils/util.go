package utils

import (
	"bytes"
	"encoding/gob"
	"github.com/gotd/td/tg"
	"strconv"
)

func init() {
	gob.Register(&tg.InputFile{})
}

func ToString(data any) string {
	switch data.(type) {

	case int:
		return strconv.Itoa(data.(int))
	case int64:
		return strconv.FormatInt(data.(int64), 10)
	case float64:
		return strconv.FormatFloat(data.(float64), 'E', -1, 64)
	default:
		return ""

	}
}

func ToInt(data any) int {
	switch data.(type) {

	case int:
		return data.(int)
	case string:
		i, err := strconv.Atoi(data.(string))
		if err != nil {
			return 0
		}
		return i
	default:
		return 0
	}
}

func ToInt64(data any) int64 {
	switch data.(type) {

	case int:
		return int64(data.(int))
	case string:
		i, err := strconv.ParseInt(data.(string), 10, 64)
		if err != nil {
			return 0
		}
		return i
	default:
		return 0
	}
}

func EncodeObject(p any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf) // 构造编码器，并把数据写进buf中
	if err := encoder.Encode(p); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeObject[T any](b []byte, obj *T) error {
	//var buf bytes.Buffer
	bufPtr := bytes.NewBuffer(b)      // 返回的类型是 *Buffer，而不是 Buffer。注意一下
	decoder := gob.NewDecoder(bufPtr) // 从 bufPtr 中获取数据

	if err := decoder.Decode(obj); err != nil { // 将数据写进变量 p 中
		return err
	}
	return nil
}
