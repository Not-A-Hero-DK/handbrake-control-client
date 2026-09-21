# HandBrake Control

Native macOS worker agent, menu-bar companion, and future Unraid dashboard for
the HandBrake automation workflow.

## Contents

- `cmd/hb-agent`: Apple Silicon Go agent entry point.
- `internal`: agent state and its local Unix-socket control API.
- `menu`: native AppKit menu-bar companion.
- `config`: safe default configuration packaged into the app on first launch.
- `docs`: architecture, installation, and API notes.

This public repository is self-contained; do not commit personal conversion
scripts, media paths, mounted-share details, or local configuration.

## Current checks

Prerequisites for a macOS build are Go (the version in `go.mod`) and Xcode
Command Line Tools. `HandBrakeCLI` is optional for building, but required for
the agent to report that encoding is ready. Install it through the official
HandBrake distribution or a package manager such as Homebrew.

```sh
GOCACHE=/private/tmp/handbrake-control-go-cache go test ./...
GOCACHE=/private/tmp/handbrake-control-go-cache go build -o bin/hb-agent ./cmd/hb-agent
```

## Run the macOS prototype

Build the Finder-launchable app bundle:

```sh
./scripts/build-macos-app.sh
open "dist/HandBrake Control.app"
```

It appears as a film icon in the macOS menu bar and starts the bundled local
agent. Its status line verifies the HandBrake executable and SMB mount
state. Use **Settings…** to save the current worker and power preferences
locally. The current `Start worker`, `Stop and requeue`, and `Drain` items are
intentionally non-operative until the conversion-worker integration is built.

Quit the menu-bar app to stop its bundled agent.

## Continuous integration

GitHub Actions runs on every pull request and change to `main`. It checks Go
formatting, runs tests and `go vet`, validates the packaged default
configuration, builds the macOS app, checks its bundle contents, and verifies
its ad-hoc signature. CI confirms build and local-readiness behavior; it does
not run real conversions, mount SMB shares, or alter macOS power settings.

## Documentation

- [Architecture](docs/architecture.md)
- [macOS agent installation](docs/agent-install.md)
- [Local agent control API](docs/local-control-api.md)
- [Menu-bar companion](menu/README.md)
- [Product and release plan](docs/PLAN.md)
