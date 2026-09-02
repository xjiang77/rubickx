package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xjiang77/rubickx/network-security/internal/evidence"
	"github.com/xjiang77/rubickx/network-security/internal/lab"
)

func main() {
	labID := flag.String("lab", "all", "lab id or all")
	output := flag.String("out", "evidence/all.json", "JSON evidence output")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var report evidence.Report
	var err error
	if *labID == "all" {
		report, err = lab.RunAll(ctx)
	} else {
		var result evidence.LabResult
		result, err = lab.Run(ctx, *labID)
		report = evidence.Report{SchemaVersion: 2, Labs: []evidence.LabResult{result}}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report.GeneratedAt = time.Now().UTC().Format(time.RFC3339Nano)
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, append(encoded, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %d lab results to %s\n", len(report.Labs), *output)
}
