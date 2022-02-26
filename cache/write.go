package cache

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

func writeJSON(key *Key) (string, error) {
	js, err := json.Marshal(key)
	if err != nil {
		return "", err
	}
	f, err := ioutil.TempFile("", "zim-key-")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(js); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func writeKey(path string, key *Key) error {
	js, err := json.Marshal(key)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(js)
	return err
}
