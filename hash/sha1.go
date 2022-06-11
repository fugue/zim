package hash

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
)

type sha1Hasher struct{}

func SHA1() Hasher {
	return &sha1Hasher{}
}

func (hasher *sha1Hasher) Object(obj interface{}) (string, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	h := sha1.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (hasher *sha1Hasher) File(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	h := sha1.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (hasher *sha1Hasher) String(s string) (string, error) {
	h := sha1.New()
	if _, err := h.Write([]byte(s)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
