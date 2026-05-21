package checker

import (
	"context"
	"fmt"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

type ProcessChecker struct {
	name        string
	processName string
}

func NewProcessChecker(name string, options map[string]interface{}) *ProcessChecker {
	processName := ""
	if v, ok := options["process_name"]; ok {
		if s, ok := v.(string); ok {
			processName = s
		}
	}
	return &ProcessChecker{name: name, processName: processName}
}

func (c *ProcessChecker) Name() string { return c.name }
func (c *ProcessChecker) Type() string { return "process" }

func (c *ProcessChecker) Check(ctx context.Context) CheckResult {
	labels := map[string]string{"process_name": c.processName}

	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return errorResult(c.name, c.Type(), fmt.Sprintf("failed to list processes: %v", err))
	}

	var matchCount int
	for _, p := range processes {
		pname, err := p.NameWithContext(ctx)
		if err != nil {
			continue
		}
		if strings.Contains(pname, c.processName) {
			matchCount++
		}
	}

	running := 0.0
	if matchCount > 0 {
		running = 1.0
	}

	metrics := []Metric{
		{Name: "running", Value: running, Unit: "bool"},
		{Name: "pid_count", Value: float64(matchCount), Unit: "count"},
	}

	if matchCount == 0 {
		result := errorResult(c.name, c.Type(), fmt.Sprintf("process %q is not running", c.processName))
		result.Metrics = metrics
		result.Labels = labels
		return result
	}

	return okResult(c.name, c.Type(), fmt.Sprintf("process %q is running (%d PIDs)", c.processName, matchCount), metrics, labels)
}
