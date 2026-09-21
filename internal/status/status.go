package status

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Not-A-Hero-DK/handbrake-control/internal/config"
)

type Report struct {
	AgentID        string       `json:"agentId"`
	DisplayName    string       `json:"displayName"`
	CollectedAt    time.Time    `json:"collectedAt"`
	Platform       string       `json:"platform"`
	Architecture   string       `json:"architecture"`
	HandBrakeReady bool         `json:"handbrakeReady"`
	Mounts         []MountState `json:"mounts"`
	WorkerEnabled  bool         `json:"workerEnabled"`
}

type MountState struct {
	Name       string `json:"name"`
	MountPoint string `json:"mountPoint"`
	Mounted    bool   `json:"mounted"`
}

func Collect(cfg config.Config) (Report, error) {
	report := Report{
		AgentID:        cfg.Agent.ID,
		DisplayName:    cfg.Agent.DisplayName,
		CollectedAt:    time.Now().UTC(),
		Platform:       runtime.GOOS,
		Architecture:   runtime.GOARCH,
		HandBrakeReady: handBrakeAvailable(),
		WorkerEnabled:  cfg.Worker.Enabled,
	}
	for _, profile := range cfg.MountProfiles {
		report.Mounts = append(report.Mounts, MountState{
			Name: profile.Name, MountPoint: profile.MountPoint, Mounted: mounted(profile.MountPoint),
		})
	}
	return report, nil
}

func executable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0111 != 0
}

func handBrakeAvailable() bool {
	if path, err := exec.LookPath("HandBrakeCLI"); err == nil && executable(path) {
		return true
	}
	for _, path := range []string{
		"/opt/homebrew/bin/HandBrakeCLI",
		"/usr/local/bin/HandBrakeCLI",
		"/Applications/HandBrake.app/Contents/MacOS/HandBrakeCLI",
	} {
		if executable(filepath.Clean(path)) {
			return true
		}
	}
	return false
}

func mounted(path string) bool {
	output, err := exec.Command("mount").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), " on "+path+" ")
}
