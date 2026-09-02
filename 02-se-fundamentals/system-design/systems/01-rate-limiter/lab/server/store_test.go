package server

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryFixedWindowStore(t *testing.T) {
	t.Parallel()

	store := NewMemoryFixedWindowStore()
	now := time.UnixMilli(1_200)
	first, err := store.Allow(context.Background(), "alice", 1, time.Second, 1, now)
	if err != nil || !first.Allowed {
		t.Fatalf("first Allow() = %+v, %v", first, err)
	}
	second, err := store.Allow(context.Background(), "alice", 1, time.Second, 1, now)
	if err != nil || second.Allowed || second.RetryAfterMs != 800 {
		t.Fatalf("second Allow() = %+v, %v; want rejected with 800ms retry", second, err)
	}
}

func TestMemoryFixedWindowStoreAllowsExactlyLimitUnderConcurrency(t *testing.T) {
	t.Parallel()
	store := NewMemoryFixedWindowStore()
	now := time.UnixMilli(1_200)
	var allowed atomic.Int64
	var wait sync.WaitGroup
	for i := 0; i < 20; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			decision, err := store.Allow(context.Background(), "alice", 5, time.Second, 1, now)
			if err != nil {
				t.Errorf("Allow() error = %v", err)
				return
			}
			if decision.Allowed {
				allowed.Add(1)
			}
		}()
	}
	wait.Wait()
	if allowed.Load() != 5 {
		t.Fatalf("allowed = %d, want exactly 5", allowed.Load())
	}
	probe, err := store.Allow(context.Background(), "alice", 6, time.Second, 1, now)
	if err != nil || !probe.Allowed {
		t.Fatalf("denied requests changed stored usage: probe=%+v err=%v", probe, err)
	}
}

func TestRedisFixedWindowStoreUsesAtomicLua(t *testing.T) {
	if _, err := exec.LookPath("redis-server"); err != nil {
		t.Skip("redis-server is not installed")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("redis-server", "--bind", "127.0.0.1", "--port", strconv.Itoa(port), "--save", "", "--appendonly", "no")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start redis-server: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(3 * time.Second)
	for {
		conn, dialErr := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("redis-server did not become ready: %v", dialErr)
		}
		time.Sleep(20 * time.Millisecond)
	}

	store := NewRedisFixedWindowStore(addr, "rate-limiter-test:")
	first, err := store.Allow(context.Background(), "alice", 1, time.Second, 1, time.UnixMilli(1_200))
	if err != nil || !first.Allowed {
		t.Fatalf("first Redis Allow() = %+v, %v", first, err)
	}
	second, err := store.Allow(context.Background(), "alice", 1, time.Second, 1, time.UnixMilli(1_200))
	if err != nil || second.Allowed || second.RetryAfterMs < 650 || second.RetryAfterMs > 800 {
		t.Fatalf("second Redis Allow() = %+v, %v; want rejected near the 800ms window boundary", second, err)
	}

	var allowed atomic.Int64
	var wait sync.WaitGroup
	errors := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			decision, allowErr := store.Allow(context.Background(), "concurrent", 5, time.Second, 1, time.UnixMilli(1_200))
			if allowErr != nil {
				errors <- allowErr
				return
			}
			if decision.Allowed {
				allowed.Add(1)
			}
		}()
	}
	wait.Wait()
	close(errors)
	for allowErr := range errors {
		t.Errorf("concurrent Allow() error = %v", allowErr)
	}
	if allowed.Load() != 5 {
		t.Fatalf("concurrent allowed = %d, want exactly 5", allowed.Load())
	}
	probe, err := store.Allow(context.Background(), "concurrent", 6, time.Second, 1, time.UnixMilli(1_200))
	if err != nil || !probe.Allowed {
		t.Fatalf("denied requests changed stored usage: probe = %+v, %v", probe, err)
	}

	impossible, err := store.Allow(context.Background(), "impossible", 5, time.Second, 7, time.UnixMilli(1_200))
	if err != nil || impossible.Allowed || impossible.RetryAfterMs != 0 || impossible.Reason != "cost_exceeds_limit" {
		t.Fatalf("impossible cost = %+v, %v", impossible, err)
	}
	afterImpossible, err := store.Allow(context.Background(), "impossible", 1, time.Second, 1, time.UnixMilli(1_200))
	if err != nil || !afterImpossible.Allowed {
		t.Fatalf("impossible cost created or incremented Redis key: %+v, %v", afterImpossible, err)
	}
}
