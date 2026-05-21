package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"ops-worker/checker"
)

type ReportPayload struct {
	Hostname string             `json:"hostname"`
	SentAt   time.Time          `json:"sent_at"`
	Result   checker.CheckResult `json:"result"`
}

type Sender struct {
	url      string
	password string
	client   *http.Client
}

func New(url, password string, timeoutSec int) *Sender {
	return &Sender{
		url:      url,
		password: password,
		client: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
	}
}

func (s *Sender) Send(ctx context.Context, result checker.CheckResult) error {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	payload := ReportPayload{
		Hostname: hostname,
		SentAt:   time.Now().UTC(),
		Result:   result,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
