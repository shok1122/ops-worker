package checker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type DockerChecker struct {
	name          string
	containerName string
}

func NewDockerChecker(name string, options map[string]interface{}) *DockerChecker {
	containerName := ""
	if v, ok := options["container_name"]; ok {
		if s, ok := v.(string); ok {
			containerName = s
		}
	}
	return &DockerChecker{name: name, containerName: containerName}
}

func (c *DockerChecker) Name() string { return c.name }
func (c *DockerChecker) Type() string { return "docker" }

func (c *DockerChecker) Check(ctx context.Context) CheckResult {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return errorResult(c.name, c.Type(), fmt.Sprintf("failed to create Docker client: %v", err))
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return errorResult(c.name, c.Type(), fmt.Sprintf("failed to list containers: %v", err))
	}

	for _, ctr := range containers {
		for _, cname := range ctr.Names {
			// Docker prepends "/" to container names
			normalizedName := cname
			if len(normalizedName) > 0 && normalizedName[0] == '/' {
				normalizedName = normalizedName[1:]
			}
			if normalizedName == c.containerName {
				running := 0.0
				if ctr.State == "running" {
					running = 1.0
				}

				metrics := []Metric{
					{Name: "running", Value: running, Unit: "bool"},
				}
				labels := map[string]string{
					"container_name":   c.containerName,
					"container_id":     ctr.ID[:12],
					"container_status": ctr.Status,
				}

				if ctr.State != "running" {
					result := errorResult(c.name, c.Type(), fmt.Sprintf("container %q is not running (state: %s)", c.containerName, ctr.State))
					result.Metrics = metrics
					result.Labels = labels
					return result
				}

				return okResult(c.name, c.Type(), fmt.Sprintf("container %q is running", c.containerName), metrics, labels)
			}
		}
	}

	// Container not found
	metrics := []Metric{
		{Name: "running", Value: 0, Unit: "bool"},
	}
	labels := map[string]string{
		"container_name":   c.containerName,
		"container_id":     "",
		"container_status": "not_found",
	}
	result := errorResult(c.name, c.Type(), fmt.Sprintf("container %q not found", c.containerName))
	result.Metrics = metrics
	result.Labels = labels
	return result
}
