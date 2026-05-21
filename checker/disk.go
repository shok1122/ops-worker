package checker

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v3/disk"
)

type DiskChecker struct {
	name string
	path string
}

func NewDiskChecker(name string, options map[string]interface{}) *DiskChecker {
	path := "/"
	if v, ok := options["path"]; ok {
		if s, ok := v.(string); ok {
			path = s
		}
	}
	return &DiskChecker{name: name, path: path}
}

func (c *DiskChecker) Name() string { return c.name }
func (c *DiskChecker) Type() string { return "disk" }

func (c *DiskChecker) Check(ctx context.Context) CheckResult {
	usage, err := disk.UsageWithContext(ctx, c.path)
	if err != nil {
		return errorResult(c.name, c.Type(), fmt.Sprintf("failed to get disk usage for %s: %v", c.path, err))
	}

	const toGB = 1.0 / (1024 * 1024 * 1024)
	metrics := []Metric{
		{Name: "total_gb", Value: float64(usage.Total) * toGB, Unit: "GB"},
		{Name: "used_gb", Value: float64(usage.Used) * toGB, Unit: "GB"},
		{Name: "free_gb", Value: float64(usage.Free) * toGB, Unit: "GB"},
		{Name: "usage_percent", Value: usage.UsedPercent, Unit: "percent"},
	}

	labels := map[string]string{"path": c.path}

	return okResult(c.name, c.Type(), fmt.Sprintf("Disk usage at %s: %.1f%%", c.path, usage.UsedPercent), metrics, labels)
}
