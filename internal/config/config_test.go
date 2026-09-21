package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExpandsHomePlaceholderAndTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "agent.json")
	contents := `{
  "agent": {"id":"test", "displayName":"Test", "heartbeatSeconds":5, "socketPath":"${HOME}/agent.sock"},
  "server": {"url":"https://example.test", "enrollmentToken":"local"},
  "worker": {"workDirectory":"~/handbrake-control", "enabled":false},
  "mountProfiles": [{"name":"media", "shareURL":"//guest@example.test/media", "mountPoint":"${HOME}/Volumes/Media", "credentialKeychainService":"test"}],
  "power": {"stopIfBatteryBelowPercent":70}
}`
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Agent.SocketPath != filepath.Join(home, "agent.sock") {
		t.Fatalf("socket path = %q", cfg.Agent.SocketPath)
	}
	if cfg.Worker.WorkDirectory != filepath.Join(home, "handbrake-control") {
		t.Fatalf("work directory = %q", cfg.Worker.WorkDirectory)
	}
	if err := cfg.ValidateLocal(); err != nil {
		t.Fatalf("ValidateLocal() error = %v", err)
	}
}
