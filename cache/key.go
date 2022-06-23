package cache

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
)

// Key contains information used to build a key
type Key struct {
	Project     string   `json:"project"`
	Component   string   `json:"component"`
	Rule        string   `json:"rule"`
	Image       string   `json:"image"`
	OutputCount int      `json:"output_count"`
	Inputs      []*Entry `json:"inputs"`
	Deps        []*Entry `json:"deps"`
	Env         []*Entry `json:"env"`
	Toolchain   []*Entry `json:"toolchain"`
	Version     string   `json:"version"`
	Commands    []string `json:"commands"`
	Native      bool     `json:"native,omitempty"`
	hex         string
}

// String returns the key as a hexadecimal string
func (k *Key) String() string {
	return k.hex
}

// Compute determines the hash for this key
func (k *Key) Compute() error {
	h := sha1.New()
	if err := json.NewEncoder(h).Encode(k); err != nil {
		return err
	}
	k.hex = hex.EncodeToString(h.Sum(nil))
	return nil
}
