package integration_test

import (
	"context"
	"os/exec"
	"testing"
)

func requireDocker(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5_000_000_000)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skipf("Docker not available, skipping integration test: %v", err)
	}
}
