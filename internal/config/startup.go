package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Startup contains the login settings supplied by the Nix module's wrapper.
type Startup struct {
	AutoStart   bool `json:"auto_start"`
	SilentStart bool `json:"silent_start"`
}

// NixStartup returns nil when startup is managed by the application.
func NixStartup() (*Startup, error) {
	value := os.Getenv("BITERNET_NIX_STARTUP")
	if value == "" {
		return nil, nil
	}

	var startup Startup
	if err := json.Unmarshal([]byte(value), &startup); err != nil {
		return nil, fmt.Errorf("读取 Nix 启动设置: %w", err)
	}
	return &startup, nil
}
