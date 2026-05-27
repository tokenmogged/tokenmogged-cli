package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/tokenmogged/tokenmogged-cli/internal/config"
)

type ActiveMatch struct {
	MatchID    string    `json:"match_id"`
	MatchToken string    `json:"match_token"`
	PlayerSide string    `json:"player_side"`
	State      string    `json:"state"`
	Mode       string    `json:"mode"`
	TopicID    string    `json:"topic_id"`
	StartedAt  time.Time `json:"started_at"`
	RefreshedAt time.Time `json:"refreshed_at"`
}

func path() (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "active_match.json"), nil
}

func LoadActiveMatch() (*ActiveMatch, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var m ActiveMatch
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.MatchID == "" || m.MatchToken == "" {
		return nil, nil
	}
	if time.Since(m.RefreshedAt) > 70*time.Minute {
		return nil, nil
	}
	return &m, nil
}

func SaveActiveMatch(m *ActiveMatch) error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	m.RefreshedAt = time.Now()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

func ClearActiveMatch() error {
	p, err := path()
	if err != nil {
		return err
	}
	err = os.Remove(p)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
