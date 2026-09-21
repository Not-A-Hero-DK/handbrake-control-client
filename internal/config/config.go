package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Agent struct {
		ID               string `json:"id"`
		DisplayName      string `json:"displayName"`
		HeartbeatSeconds int    `json:"heartbeatSeconds"`
		SocketPath       string `json:"socketPath"`
	} `json:"agent"`
	Server struct {
		URL             string `json:"url"`
		EnrollmentToken string `json:"enrollmentToken"`
	} `json:"server"`
	Worker struct {
		WorkDirectory string `json:"workDirectory"`
		Enabled       bool   `json:"enabled"`
	} `json:"worker"`
	MountProfiles []MountProfile `json:"mountProfiles"`
	Power         PowerPolicy    `json:"power"`
}

type MountProfile struct {
	Name                      string `json:"name"`
	ShareURL                  string `json:"shareURL"`
	MountPoint                string `json:"mountPoint"`
	CredentialKeychainService string `json:"credentialKeychainService"`
}

type PowerPolicy struct {
	AllowClosedLidWork        bool `json:"allowClosedLidWork"`
	RequireACPower            bool `json:"requireACPower"`
	StopIfBatteryBelowPercent int  `json:"stopIfBatteryBelowPercent"`
	RestoreSleepOnIdle        bool `json:"restoreSleepOnIdle"`
}

func Load(path string) (Config, error) {
	var cfg Config
	contents, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(contents, &cfg); err != nil {
		return cfg, err
	}
	cfg.expandPaths()
	return cfg, nil
}

func (c *Config) expandPaths() {
	c.Agent.SocketPath = expandPath(c.Agent.SocketPath)
	c.Worker.WorkDirectory = expandPath(c.Worker.WorkDirectory)
	for index := range c.MountProfiles {
		c.MountProfiles[index].MountPoint = expandPath(c.MountProfiles[index].MountPoint)
	}
}

func expandPath(value string) string {
	if home, err := os.UserHomeDir(); err == nil {
		value = strings.ReplaceAll(value, "${HOME}", home)
		if value == "~" {
			value = home
		} else if strings.HasPrefix(value, "~/") {
			value = filepath.Join(home, strings.TrimPrefix(value, "~/"))
		}
	}
	return os.ExpandEnv(value)
}

func (c Config) Validate() error {
	if err := c.ValidateLocal(); err != nil {
		return err
	}
	if strings.TrimSpace(c.Server.EnrollmentToken) == "" || c.Server.EnrollmentToken == "replace-this-locally" {
		return fmt.Errorf("server.enrollmentToken must be set locally")
	}
	return nil
}

// ValidateLocal checks the configuration needed before the agent begins making
// network requests. This lets the local status socket work during enrollment.
func (c Config) ValidateLocal() error {
	if strings.TrimSpace(c.Agent.ID) == "" {
		return fmt.Errorf("agent.id is required")
	}
	if c.Agent.HeartbeatSeconds < 5 {
		return fmt.Errorf("agent.heartbeatSeconds must be at least 5")
	}
	if c.Agent.SocketPath != "" && !filepath.IsAbs(c.Agent.SocketPath) {
		return fmt.Errorf("agent.socketPath must be absolute when set")
	}
	if !strings.HasPrefix(c.Server.URL, "https://") {
		return fmt.Errorf("server.url must use https")
	}
	if !filepath.IsAbs(c.Worker.WorkDirectory) {
		return fmt.Errorf("worker.workDirectory must be absolute")
	}
	if c.Power.StopIfBatteryBelowPercent < 0 || c.Power.StopIfBatteryBelowPercent > 100 {
		return fmt.Errorf("power.stopIfBatteryBelowPercent must be between 0 and 100")
	}
	for _, profile := range c.MountProfiles {
		if profile.Name == "" || !strings.HasPrefix(profile.ShareURL, "//") || !filepath.IsAbs(profile.MountPoint) {
			return fmt.Errorf("each mount profile requires name, // shareURL, and absolute mountPoint")
		}
	}
	return nil
}
