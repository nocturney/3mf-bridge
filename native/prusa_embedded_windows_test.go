package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func bundledPrusaFixture(t *testing.T) string {
	t.Helper()
	var payload bytes.Buffer
	archive := zip.NewWriter(&payload)
	for _, entry := range []struct{ name, contents string }{
		{"slicebridge-prusa-engine.exe", "MZexample-sidecar-binary"},
		{"panda2prusa-LICENSE.txt", "MIT License\nPermission is hereby granted, free of charge"},
	} {
		writer, err := archive.Create(entry.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(entry.contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(t.TempDir(), "SliceBridge.exe")
	var bin bytes.Buffer
	bin.WriteString("MZexample-main-program")
	bin.Write(payload.Bytes())
	bin.WriteString(prusaBundleMagic)
	if err := binary.Write(&bin, binary.LittleEndian, uint64(payload.Len())); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(payload.Bytes())
	bin.Write(digest[:])
	if err := os.WriteFile(app, bin.Bytes(), 0o700); err != nil {
		t.Fatal(err)
	}
	return app
}

func TestEmbeddedWindowsPrusaExtractionAndRecovery(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	app := bundledPrusaFixture(t)
	engine, present, err := locateEmbeddedPrusaEngine(app)
	if err != nil || !present {
		t.Fatalf("embedded payload unavailable: %q %v %v", engine, present, err)
	}
	got, err := os.ReadFile(engine)
	if err != nil || string(got) != "MZexample-sidecar-binary" {
		t.Fatalf("extracted engine mismatch: %q %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(engine), "panda2prusa-LICENSE.txt")); err != nil {
		t.Fatalf("embedded MIT license not extracted: %v", err)
	}
	if err := os.WriteFile(engine, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	recovered, present, err := locateEmbeddedPrusaEngine(app)
	if err != nil || !present || recovered != engine {
		t.Fatalf("tampered cache was not recovered: %q %v %v", recovered, present, err)
	}
	got, err = os.ReadFile(engine)
	if err != nil || string(got) != "MZexample-sidecar-binary" {
		t.Fatalf("engine not restored to embedded bytes: %q %v", got, err)
	}
}

func TestEmbeddedWindowsPrusaMissingAndTamperedPayload(t *testing.T) {
	plain := filepath.Join(t.TempDir(), "plain.exe")
	if err := os.WriteFile(plain, []byte("MZplain-app"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, present, err := locateEmbeddedPrusaEngine(plain)
	if present || err != nil {
		t.Fatalf("plain Wails developer build must use legacy lookup: %v %v", present, err)
	}
	app := bundledPrusaFixture(t)
	raw, err := os.ReadFile(app)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)-1] ^= 0xFF
	if err := os.WriteFile(app, raw, 0o700); err != nil {
		t.Fatal(err)
	}
	_, present, err = locateEmbeddedPrusaEngine(app)
	if !present || err == nil {
		t.Fatalf("checksum damage must fail closed: %v %v", present, err)
	}
}
