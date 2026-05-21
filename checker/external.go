package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

type ExternalChecker struct {
	name    string
	command string
	args    []string
	timeout int
}

func NewExternalChecker(name string, options map[string]interface{}) *ExternalChecker {
	command := ""
	if v, ok := options["command"]; ok {
		if s, ok := v.(string); ok {
			command = s
		}
	}

	var args []string
	if v, ok := options["args"]; ok {
		switch val := v.(type) {
		case []interface{}:
			for _, a := range val {
				if s, ok := a.(string); ok {
					args = append(args, s)
				}
			}
		case []string:
			args = val
		}
	}

	timeout := 5
	if v, ok := options["timeout"]; ok {
		switch val := v.(type) {
		case int:
			timeout = val
		case float64:
			timeout = int(val)
		}
	}

	return &ExternalChecker{
		name:    name,
		command: command,
		args:    args,
		timeout: timeout,
	}
}

func (c *ExternalChecker) Name() string { return c.name }
func (c *ExternalChecker) Type() string { return "external" }

type externalOutput struct {
	Status  string            `json:"status"`
	Message string            `json:"message"`
	Metrics []Metric          `json:"metrics"`
	Labels  map[string]string `json:"labels"`
}

func (c *ExternalChecker) Check(ctx context.Context) CheckResult {
	if c.command == "" {
		return errorResult(c.name, c.Type(), "command option is required")
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.command, c.args...)
	out, err := cmd.Output()

	baseLabels := map[string]string{"command": c.command}

	if err != nil {
		msg := fmt.Sprintf("command failed: %v", err)
		result := errorResult(c.name, c.Type(), msg)
		result.Labels = baseLabels
		return result
	}

	var output externalOutput
	if err := json.Unmarshal(out, &output); err != nil {
		msg := fmt.Sprintf("failed to parse command output as JSON: %v", err)
		result := errorResult(c.name, c.Type(), msg)
		result.Labels = baseLabels
		return result
	}

	// Merge labels with command key
	if output.Labels == nil {
		output.Labels = make(map[string]string)
	}
	output.Labels["command"] = c.command

	if output.Metrics == nil {
		output.Metrics = []Metric{}
	}

	status := output.Status
	if status == "" {
		status = "ok"
	}

	return CheckResult{
		Name:      c.name,
		Type:      c.Type(),
		Timestamp: time.Now(),
		Status:    status,
		Message:   output.Message,
		Metrics:   output.Metrics,
		Labels:    output.Labels,
	}
}
