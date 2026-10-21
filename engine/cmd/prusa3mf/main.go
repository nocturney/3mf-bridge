package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/nocturney/slicebridge/engine/prusa3mf"
)

func main() {
	source := flag.String("source", "", "Source Bambu/Orca project .3mf")
	output := flag.String("output", "", "Destination PrusaSlicer 3MF (never overwritten)")
	flag.Parse()
	if *source == "" || *output == "" {
		emit(map[string]any{"ok": false, "code": "ARGS_INVALID", "error": "--source and --output are required"})
		os.Exit(2)
	}
	report, err := prusa3mf.Convert(*source, *output)
	if err != nil {
		emit(map[string]any{"ok": false, "code": prusa3mf.ErrorCode(err), "error": err.Error()})
		os.Exit(1)
	}
	emit(map[string]any{"ok": true, "report": report})
}

func emit(v any) {
	if err := json.NewEncoder(os.Stdout).Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
