package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/nocturney/3mf-bridge/engine/bridge"
)

func main() {
	source := flag.String("source", "", "Source Bambu project .3mf")
	output := flag.String("output", "", "Destination 3MF (never overwritten)")
	target := flag.String("target", "u1", "Target adapter")
	flag.Parse()
	if *source == "" || *output == "" {
		emit(map[string]any{"ok": false, "code": "ARGS_INVALID", "error": "--source and --output are required"})
		os.Exit(2)
	}
	result, err := bridge.Convert(*source, *output, *target)
	if err != nil {
		emit(map[string]any{"ok": false, "code": bridge.ErrorCode(err), "error": err.Error()})
		os.Exit(1)
	}
	emit(result)
}

func emit(v any) {
	if err := json.NewEncoder(os.Stdout).Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
