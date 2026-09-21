package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Not-A-Hero-DK/handbrake-control-client/internal/config"
	"github.com/Not-A-Hero-DK/handbrake-control-client/internal/status"
)

type State string

const (
	StateDisabled State = "disabled"
	StateReady    State = "ready"
	StateDraining State = "draining"
	StatePaused   State = "paused"
)

type Runtime struct {
	mu      sync.RWMutex
	config  config.Config
	state   State
	message string
}

type Snapshot struct {
	status.Report
	State   State  `json:"state"`
	Message string `json:"message"`
}

type Command struct {
	Action string `json:"action"`
}

func New(cfg config.Config) *Runtime {
	state := StateDisabled
	message := "Worker is disabled"
	if cfg.Worker.Enabled {
		state = StateReady
		message = "Waiting for mount validation before claiming work"
	}
	return &Runtime{config: cfg, state: state, message: message}
}

func (r *Runtime) Snapshot() (Snapshot, error) {
	report, err := status.Collect(r.config)
	if err != nil {
		return Snapshot{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	report.WorkerEnabled = r.state != StateDisabled && r.state != StatePaused
	return Snapshot{Report: report, State: r.state, Message: r.message}, nil
}

func (r *Runtime) Command(command Command) (Snapshot, error) {
	r.mu.Lock()
	switch command.Action {
	case "enable":
		r.state = StateReady
		r.message = "Enabled; mount validation is required before claiming work"
	case "disable":
		r.state = StateDisabled
		r.message = "Disabled; no new jobs will be claimed"
	case "drain":
		r.state = StateDraining
		r.message = "Will stop after the current job; no new job will be claimed"
	case "stop":
		r.state = StatePaused
		r.message = "Stop and requeue requested; no active conversion is running"
	default:
		r.mu.Unlock()
		return Snapshot{}, fmt.Errorf("unsupported action %q", command.Action)
	}
	r.mu.Unlock()
	return r.Snapshot()
}

func (r *Runtime) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/status", r.handleStatus)
	mux.HandleFunc("POST /v1/commands", r.handleCommand)
	return mux
}

func (r *Runtime) handleStatus(w http.ResponseWriter, _ *http.Request) {
	snapshot, err := r.Snapshot()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (r *Runtime) handleCommand(w http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	var command Command
	decoder := json.NewDecoder(http.MaxBytesReader(w, request.Body, 4<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&command); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid command: %w", err))
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid command: request body must contain one JSON object"))
		return
	}
	snapshot, err := r.Command(command)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

// ServeLocal exposes the agent only through a Unix-domain socket. The socket
// mode prevents other local users from reading status or submitting commands.
func ServeLocal(ctx context.Context, socketPath string, handler http.Handler) error {
	return serveLocal(ctx, socketPath, handler, 5*time.Second)
}
