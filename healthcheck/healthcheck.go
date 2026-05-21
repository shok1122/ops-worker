package healthcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"ops-worker/version"
)

type AgentInfo struct {
	Version       string    `json:"version"`
	UptimeSeconds float64   `json:"uptime_seconds"`
	StartedAt     time.Time `json:"started_at"`
	GoVersion     string    `json:"go_version"`
	OS            string    `json:"os"`
	Arch          string    `json:"arch"`
}

type HealthPayload struct {
	Type      string    `json:"type"`
	Hostname  string    `json:"hostname"`
	SentAt    time.Time `json:"sent_at"`
	Agent     AgentInfo `json:"agent"`
}

type HealthChecker struct {
	url       string
	password  string
	startedAt time.Time
	client    *http.Client
}

func New(url, password string, timeoutSec int) *HealthChecker {
	return &HealthChecker{
		url:       url,
		password:  password,
		startedAt: time.Now(),
		client: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
	}
}

func (h *HealthChecker) Send(ctx context.Context) error {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	now := time.Now().UTC()
	payload := HealthPayload{
		Type:     "healthcheck",
		Hostname: hostname,
		SentAt:   now,
		Agent: AgentInfo{
			Version:       version.Version,
			UptimeSeconds: now.Sub(h.startedAt).Seconds(),
			StartedAt:     h.startedAt.UTC(),
			GoVersion:     runtime.Version(),
			OS:            runtime.GOOS,
			Arch:          runtime.GOARCH,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling healthcheck payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating healthcheck request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+h.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending healthcheck: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
