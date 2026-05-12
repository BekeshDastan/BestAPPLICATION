package integration_test

import (
	"context"
	"os/exec"
	"testing"
)

// requireDocker skips the test if the Docker daemon is not reachable.
// This prevents panics from testcontainers-go when Docker Desktop is not running.
func requireDocker(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5_000_000_000) // 5s
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skipf("Docker not available, skipping integration test: %v", err)
	}
}
