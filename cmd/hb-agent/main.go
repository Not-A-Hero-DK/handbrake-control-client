package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/Not-A-Hero-DK/handbrake-control/internal/agent"
	"github.com/Not-A-Hero-DK/handbrake-control/internal/config"
	"github.com/Not-A-Hero-DK/handbrake-control/internal/status"
)

func main() {
	configPath := flag.String("config", "agent.local.json", "path to agent JSON configuration")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "configuration error:", err)
		os.Exit(2)
	}

	if err := cfg.ValidateLocal(); err != nil {
		fmt.Fprintln(os.Stderr, "configuration error:", err)
		os.Exit(2)
	}

	command := flag.Arg(0)
	if command == "daemon" {
		socketPath, err := localSocketPath(cfg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "agent error:", err)
			os.Exit(1)
		}
		runtime := agent.New(cfg)
		fmt.Fprintln(os.Stderr, "local agent socket:", socketPath)
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := agent.ServeLocal(ctx, socketPath, runtime.Handler()); err != nil {
			fmt.Fprintln(os.Stderr, "agent server error:", err)
			os.Exit(1)
		}
		return
	}

	if command != "" && command != "status" {
		fmt.Fprintln(os.Stderr, "usage: hb-agent [-config path] [status|daemon]")
		os.Exit(2)
	}

	report, err := status.Collect(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "status error:", err)
		os.Exit(1)
	}

	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, "encoding status:", err)
		os.Exit(1)
	}
}

func localSocketPath(cfg config.Config) (string, error) {
	if cfg.Agent.SocketPath != "" {
		return cfg.Agent.SocketPath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", "HandBrake Control", "agent.sock"), nil
}
