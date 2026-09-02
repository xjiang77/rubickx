package server

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJavaRunnerCompileAndRunDeadlines(t *testing.T) {
	for _, test := range []struct {
		name          string
		javaScript    string
		parentTimeout time.Duration
		wantError     string
	}{
		{name: "slow compile keeps full run budget", javaScript: `printf '{"events":[],"decisions":[]}'`},
		{name: "parent deadline cancels compile", parentTimeout: 100 * time.Millisecond, wantError: "compile Java runner"},
		{name: "run still has a deadline", javaScript: "exec sleep 3", wantError: "java runner timed out"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			sources := filepath.Join(root, "runners", "java")
			if err := os.MkdirAll(sources, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(sources, "RateLimiterRunner.java"), []byte("class RateLimiterRunner {}"), 0o644); err != nil {
				t.Fatal(err)
			}
			bin := t.TempDir()
			// 用可控的慢编译复现冷缓存 CI，避免测试依赖真实 javac 的机器耗时。
			for name, script := range map[string]string{"javac": "exec sleep 2", "java": test.javaScript} {
				if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			runner := NewSubprocessRunner(LanguageJava, root)
			runner.Timeout = time.Second
			ctx := context.Background()
			if test.parentTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, test.parentTimeout)
				defer cancel()
			}
			_, err := runner.Run(ctx, RunRequest{Algorithm: AlgorithmTokenBucket, Config: map[string]float64{"capacity": 1, "ratePerSecond": 1}, RequestTimeline: []RequestPoint{}})
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("Run() after slow compile: %v", err)
				}
			} else if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("Run() error = %v, want %q wrapping context.DeadlineExceeded", err, test.wantError)
			}
		})
	}
}

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
