package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultAPIBase = "https://tokenmogged.com"

type Credentials struct {
	APIBase  string `json:"api_base"`
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func ConfigDir() (string, error) {
	if v := os.Getenv("TOKENMAXXING_CONFIG_DIR"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "tokenmogged"), nil
}

func credentialsPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

func Load() (*Credentials, error) {
	p, err := credentialsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoCredentials
		}
		return nil, err
	}
	var c Credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse credentials.json: %w", err)
	}
	if c.APIBase == "" {
		c.APIBase = DefaultAPIBase
	}
	return &c, nil
}

func Save(c *Credentials) error {
	if c.APIBase == "" {
		c.APIBase = DefaultAPIBase
	}
	p, err := credentialsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

func Clear() error {
	p, err := credentialsPath()
	if err != nil {
		return err
	}
	err = os.Remove(p)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

var ErrNoCredentials = errors.New("not logged in (run: tokenmogged login)")

func APIBase() string {
	if v := os.Getenv("TOKENMAXXING_API"); v != "" {
		return v
	}
	c, err := Load()
	if err == nil && c.APIBase != "" {
		return c.APIBase
	}
	return DefaultAPIBase
}
