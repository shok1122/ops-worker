package checker

import (
	"context"
	"time"
)

type CheckResult struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Status    string            `json:"status"`
	Message   string            `json:"message"`
	Metrics   []Metric          `json:"metrics"`
	Labels    map[string]string `json:"labels"`
	Error     string            `json:"error,omitempty"`
}

type Metric struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type Checker interface {
	Name() string
	Type() string
	Check(ctx context.Context) CheckResult
}

func errorResult(name, typ, msg string) CheckResult {
	return CheckResult{
		Name:      name,
		Type:      typ,
		Timestamp: time.Now(),
		Status:    "error",
		Message:   msg,
		Metrics:   []Metric{},
		Labels:    map[string]string{},
		Error:     msg,
	}
}

func okResult(name, typ, msg string, metrics []Metric, labels map[string]string) CheckResult {
	return CheckResult{
		Name:      name,
		Type:      typ,
		Timestamp: time.Now(),
		Status:    "ok",
		Message:   msg,
		Metrics:   metrics,
		Labels:    labels,
	}
}
