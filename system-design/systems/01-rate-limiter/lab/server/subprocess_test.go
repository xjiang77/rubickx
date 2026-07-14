package server

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSubprocessRunnerBoundsStdout(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not installed")
	}
	root := t.TempDir()
	directory := filepath.Join(root, "runners", "js")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	script := `process.stdin.resume(); process.stdin.on("end", () => process.stdout.write("x".repeat(3 * 1024 * 1024)));`
	if err := os.WriteFile(filepath.Join(directory, "runner.mjs"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := NewSubprocessRunner(LanguageJavaScript, root)
	_, err := runner.Run(context.Background(), RunRequest{Algorithm: AlgorithmTokenBucket, Config: map[string]float64{"capacity": 1, "ratePerSecond": 1}, RequestTimeline: []RequestPoint{}})
	if err == nil || !strings.Contains(err.Error(), "stdout exceeded") {
		t.Fatalf("Run() error = %v, want bounded stdout error", err)
	}
}

func TestSubprocessRunnerRejectsOversizedTimelineBeforeStartingProcess(t *testing.T) {
	runner := NewSubprocessRunner(LanguageJavaScript, t.TempDir())
	timeline := make([]RequestPoint, 101)
	for index := range timeline {
		timeline[index] = RequestPoint{AtMs: int64(index), Cost: 1, Key: "alice"}
	}
	_, err := runner.Run(context.Background(), RunRequest{Algorithm: AlgorithmTokenBucket, Config: map[string]float64{"capacity": 1, "ratePerSecond": 1}, RequestTimeline: timeline})
	if err == nil || !strings.Contains(err.Error(), "at most 100") {
		t.Fatalf("Run() error = %v, want timeline bound error", err)
	}
}
