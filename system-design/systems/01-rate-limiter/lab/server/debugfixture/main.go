package main

import (
	"encoding/json"
	"fmt"
	"os"

	lab "rubickx/system-design/systems/01-rate-limiter/lab/server"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: debug-fixture <validated-scenario.json>")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	var request lab.RunRequest
	if err := json.Unmarshal(data, &request); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := lab.RunDebugScenario(request); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
