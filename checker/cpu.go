package checker

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v3/cpu"
)

type CPUChecker struct {
	name string
}

func NewCPUChecker(name string) *CPUChecker {
	return &CPUChecker{name: name}
}

func (c *CPUChecker) Name() string { return c.name }
func (c *CPUChecker) Type() string { return "cpu" }

func (c *CPUChecker) Check(ctx context.Context) CheckResult {
	percentages, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return errorResult(c.name, c.Type(), fmt.Sprintf("failed to get CPU usage: %v", err))
	}

	counts, err := cpu.CountsWithContext(ctx, false)
	if err != nil {
		return errorResult(c.name, c.Type(), fmt.Sprintf("failed to get CPU count: %v", err))
	}

	usage := 0.0
	if len(percentages) > 0 {
		usage = percentages[0]
	}

	metrics := []Metric{
		{Name: "usage_percent", Value: usage, Unit: "percent"},
		{Name: "core_count", Value: float64(counts), Unit: "count"},
	}

	return okResult(c.name, c.Type(), fmt.Sprintf("CPU usage: %.1f%%", usage), metrics, map[string]string{})
}
