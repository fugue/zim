package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
)

type sha256Hasher struct{}

func SHA256() Hasher {
	return &sha256Hasher{}
}

func (hasher *sha256Hasher) Object(obj interface{}) (string, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (hasher *sha256Hasher) File(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (hasher *sha256Hasher) String(s string) (string, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(s)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
