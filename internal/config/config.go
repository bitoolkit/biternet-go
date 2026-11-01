package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Settings belongs to the desktop app; the client never persists credentials.
type Settings struct {
	Username         string `json:"username,omitempty"`
	Password         string `json:"password,omitempty"`
	RememberPassword bool   `json:"remember_password"`
	AutoLogin        bool   `json:"auto_login"`
	SilentStart      bool   `json:"silent_start"`
	Theme            string `json:"theme"`
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "biternet-go", "settings.json"), nil
}

func Load(path string) (Settings, error) {
	settings := Settings{Theme: "system"}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("读取设置: %w", err)
	}

	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, fmt.Errorf("解析设置: %w", err)
	}
	if settings.Theme != "system" && settings.Theme != "light" && settings.Theme != "dark" {
		return Settings{}, errors.New("无效的外观设置")
	}

	if !settings.RememberPassword {
		settings.Password = ""
		settings.AutoLogin = false
	}

	return settings, nil
}

// Save atomically replaces the file. Disabling RememberPassword removes both credentials.
func Save(path string, settings Settings) error {
	if !settings.RememberPassword {
		settings.Username = ""
		settings.Password = ""
		settings.AutoLogin = false
	}

	data, _ := json.MarshalIndent(settings, "", "  ")

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("创建设置目录: %w", err)
	}

	file, err := os.CreateTemp(dir, ".settings-*")
	if err != nil {
		return fmt.Errorf("创建设置文件: %w", err)
	}
	name := file.Name()
	defer os.Remove(name)

	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("写入设置: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("关闭设置文件: %w", err)
	}

	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("保存设置: %w", err)
	}

	return nil
}
