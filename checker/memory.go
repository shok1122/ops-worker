package checker

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v3/mem"
)

type MemoryChecker struct {
	name string
}

func NewMemoryChecker(name string) *MemoryChecker {
	return &MemoryChecker{name: name}
}

func (c *MemoryChecker) Name() string { return c.name }
func (c *MemoryChecker) Type() string { return "memory" }

func (c *MemoryChecker) Check(ctx context.Context) CheckResult {
	vmStat, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return errorResult(c.name, c.Type(), fmt.Sprintf("failed to get memory stats: %v", err))
	}

	metrics := []Metric{
		{Name: "total_bytes", Value: float64(vmStat.Total), Unit: "bytes"},
		{Name: "used_bytes", Value: float64(vmStat.Used), Unit: "bytes"},
		{Name: "available_bytes", Value: float64(vmStat.Available), Unit: "bytes"},
		{Name: "usage_percent", Value: vmStat.UsedPercent, Unit: "percent"},
	}

	return okResult(c.name, c.Type(), fmt.Sprintf("Memory usage: %.1f%%", vmStat.UsedPercent), metrics, map[string]string{})
}
