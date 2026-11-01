package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRememberAndForgetCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	want := Settings{
		Username:         " student ",
		Password:         " secret ",
		RememberPassword: true,
		AutoLogin:        true,
		Theme:            "dark",
	}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || got != want {
		t.Fatalf("round trip: %+v %v", got, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("credentials permissions: %v", info.Mode())
		}
	}

	want.RememberPassword = false
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "student") || strings.Contains(string(data), "secret") {
		t.Fatal("credentials retained")
	}
	got, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.AutoLogin || got.Password != "" || got.Username != "" || got.Theme != "dark" {
		t.Fatalf("forget: %+v", got)
	}
}

func TestMalformedSettingsAreNotSilentlyOverwritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("corrupt settings accepted")
	}
}
