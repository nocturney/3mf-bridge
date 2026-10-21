package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/nocturney/slicebridge/engine/bridge"
)

func main() {
	source := flag.String("source", "", "Source Bambu project .3mf")
	output := flag.String("output", "", "Destination 3MF (never overwritten)")
	target := flag.String("target", "u1", "Target adapter")
	profile := flag.String("profile", "", "Target vendor preset bundle")
	inspectNozzle := flag.Bool("inspect-nozzle", false, "Inspect source nozzle")
	flag.Parse()
	if *inspectNozzle {
		result, err := bridge.InspectNozzle(*source)
		if err != nil {
			emit(map[string]any{"ok": false, "code": bridge.ErrorCode(err), "error": err.Error()})
			os.Exit(1)
		}
		emit(result)
		return
	}
	if *source == "" || *output == "" {
		emit(map[string]any{"ok": false, "code": "ARGS_INVALID", "error": "--source and --output are required"})
		os.Exit(2)
	}
	result, err := bridge.ConvertWithProfile(*source, *output, *target, *profile)
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
