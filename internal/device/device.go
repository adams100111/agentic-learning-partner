package device

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Info struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	CreatedAt string `json:"createdAt"`
}

func LoadOrCreate(path string) (Info, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		var info Info
		if err := json.Unmarshal(data, &info); err != nil {
			return Info{}, fmt.Errorf("parse device identity: %w", err)
		}
		if info.ID == "" {
			return Info{}, errors.New("device identity is missing id")
		}
		return info, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Info{}, err
	}
	id, err := randomID("dev")
	if err != nil {
		return Info{}, err
	}
	info := Info{ID: id, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return Info{}, err
	}
	encoded, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return Info{}, err
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		return Info{}, err
	}
	return info, nil
}

func randomID(prefix string) (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(bytes[:]), nil
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "alp", "device.json"), nil
}
