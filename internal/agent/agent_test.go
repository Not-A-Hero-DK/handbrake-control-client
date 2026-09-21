package agent

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Not-A-Hero-DK/handbrake-control-client/internal/config"
)

func TestEnableAndDisableCommands(t *testing.T) {
	runtime := New(testConfig())

	request := httptest.NewRequest(http.MethodPost, "/v1/commands", bytes.NewBufferString(`{"action":"enable"}`))
	response := httptest.NewRecorder()
	runtime.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("enable status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/v1/commands", bytes.NewBufferString(`{"action":"disable"}`))
	response = httptest.NewRecorder()
	runtime.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("disable status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestHandlerRejectsUnknownAndTrailingCommandJSON(t *testing.T) {
	runtime := New(testConfig())
	for _, body := range []string{
		`{"action":"enable","unexpected":true}`,
		`{"action":"enable"}{"action":"disable"}`,
	} {
		request := httptest.NewRequest(http.MethodPost, "/v1/commands", bytes.NewBufferString(body))
		response := httptest.NewRecorder()
		runtime.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
		}
	}
}

func TestStatusEndpointReturnsJSON(t *testing.T) {
	runtime := New(testConfig())
	request := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	response := httptest.NewRecorder()
	runtime.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	var result Snapshot
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if result.AgentID != "test-agent" {
		t.Fatalf("agent ID = %q, want test-agent", result.AgentID)
	}
}

func testConfig() config.Config {
	var cfg config.Config
	cfg.Agent.ID = "test-agent"
	cfg.Agent.DisplayName = "Test Agent"
	cfg.Worker.WorkDirectory = "/tmp"
	return cfg
}
