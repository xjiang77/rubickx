package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/xjiang77/rubickx/network-security/internal/browserlab"
	"github.com/xjiang77/rubickx/network-security/internal/evidence"
)

type browserReport struct {
	SchemaVersion int              `json:"schema_version"`
	GeneratedAt   string           `json:"generated_at"`
	Events        []evidence.Event `json:"events"`
}

func writeEvidence(output string, events []evidence.Event) error {
	encoded, err := json.MarshalIndent(browserReport{
		SchemaVersion: 2,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Events:        events,
	}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	return os.WriteFile(output, append(encoded, '\n'), 0o644)
}

func main() {
	output := flag.String("evidence", "evidence/browser.json", "browser evidence output on shutdown")
	selfTest := flag.Bool("self-test", false, "run the loopback evidence fixture and exit")
	flag.Parse()
	server, err := browserlab.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("NETSEC_BROWSER_URL=%s\n", server.URL)
	fmt.Printf("NETSEC_ORIGIN_B_URL=%s\n", server.OriginBURL)
	if *selfTest {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := browserlab.RunEvidenceFixture(ctx, server); err != nil {
			_ = server.Close()
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := server.Close(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := writeEvidence(*output, server.Evidence()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("wrote browser evidence to %s\n", *output)
		return
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals
	if err := server.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if err := writeEvidence(*output, server.Evidence()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
