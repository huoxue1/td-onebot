package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func FileMD5(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	hash := md5.New()
	_, _ = io.Copy(hash, file)
	return hex.EncodeToString(hash.Sum(nil))
}

func FileSha256(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	hash := sha256.New()
	_, _ = io.Copy(hash, file)
	return hex.EncodeToString(hash.Sum(nil))
}

func ReaderMd5(reader io.Reader) string {
	hash := md5.New()
	_, _ = io.Copy(hash, reader)
	return hex.EncodeToString(hash.Sum(nil))
}

func strMd5(str string) (retMd5 string) {
	w := md5.New()
	io.WriteString(w, str)
	md5str := fmt.Sprintf("%x", w.Sum(nil))
	return md5str
}
