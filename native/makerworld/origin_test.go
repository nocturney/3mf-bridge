package makerworld

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOriginEvidenceIsNotGuessed(t *testing.T) {
	cases := []struct{ ref, host, kind, confidence string }{
		{"https://makerworld.com/en/models/12345", "https://public-cdn.bblmw.com/a.3mf", "makerworld", "high"},
		{"https://en.makerworld.com/abc", "https://cdn.example.org/file", "makerworld", "high"},
		{"", "https://makerworld.com/api/v1/download", "makerworld", "high"},
		{"", "https://public-cdn.bblmw.com/example.3mf", "bambu_cdn", "medium"},
		{"https://makerworld.com.evil.test/path", "https://evil.example/file.3mf", "unknown", "unknown"},
		{"https://fake-makerworld.com/models/3", "https://cdn.example.org/file", "unknown", "unknown"},
		{"https://makerworld.com@evil.test", "https://evil.test/file", "unknown", "unknown"},
		{"http://127.0.0.1/a", "http://10.0.0.1/a", "unknown", "unknown"},
		{"", "https://cdn.example.org/a?destination=makerworld.com", "unknown", "unknown"},
	}
	for _, tc := range cases {
		result := Identify(tc.ref, tc.host)
		if result.Kind != tc.kind || result.Confidence != tc.confidence {
			t.Fatalf("Identify(%q,%q) => %+v, expected %s/%s", tc.ref, tc.host, result, tc.kind, tc.confidence)
		}
	}
}

func TestWindowsZoneIdentifier(t *testing.T) {
	raw := []byte("[ZoneTransfer]\r\nZoneId=3\r\nHostUrl=https://public-cdn.bblmw.com/download/file.3mf?sig=SECRET\r\nReferrerUrl=https://makerworld.com/en/models/123-example\r\n")
	result := fromWindowsADS(raw)
	if result.Kind != "makerworld" || result.OriginURL != "https://makerworld.com/en/models/123-example" || result.Evidence != "download_referrer" {
		t.Fatalf("wrong Windows provenance: %+v", result)
	}
	if strings.Contains(result.OriginURL, "SECRET") {
		t.Fatal("signed-token leaked")
	}
	if fromWindowsADS([]byte(strings.Repeat("x", 33000))).Kind != "unknown" {
		t.Fatal("oversize ADS was parsed")
	}
}

func TestWindowsADSReadIsBounded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Zone.Identifier")
	tooLarge := []byte(strings.Repeat("x", 65536))
	if err := os.WriteFile(path, tooLarge, 0600); err != nil {
		t.Fatal(err)
	}
	data, err := readLimitedADS(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 32*1024+1 {
		t.Fatalf("oversized Windows ADS read %d bytes, wanted 32769", len(data))
	}
	if out := fromWindowsADS(data); out.Kind != "unknown" {
		t.Fatalf("truncated unsafe ADS was trusted: %+v", out)
	}
}

func TestMacWhereFrom(t *testing.T) {
	data := `(
  "https://public-cdn.bblmw.com/example.3mf?token=PRIVATE",
  "https://makerworld.com/en/models/12345"
)`
	out := fromMacSpotlight(data)
	if out.Kind != "makerworld" || out.Confidence != "high" || out.Evidence != "mac_where_from" {
		t.Fatalf("mac provenance: %+v", out)
	}
	if fromMacSpotlight("(null)").Kind != "unknown" {
		t.Fatal("missing metadata must not identify MakerWorld")
	}
}
func TestBambuDeepLinksOnWindowsAndMac(t *testing.T) {
	tests := []struct{ raw, scheme, source string }{
		{"bambustudio://open?file=https%3A%2F%2Fmakerworld.com%2Fapi%2Fdownload%2Ffile.3mf%3Fsig%3DSECRET", "bambustudio", "makerworld"},
		{"bambustudioopen://https%3A%2F%2Fpublic-cdn.bblmw.com%2Fmodel%2F123.3mf%3Ftoken%3D1", "bambustudioopen", "bambu_cdn"},
		{"slicebridge://import?url=https%3A%2F%2Fmakerworld.com%2Fen%2Fmodels%2F123", "slicebridge", "makerworld"},
	}
	for _, tc := range tests {
		link, err := ParseLink(tc.raw)
		if err != nil || link.Scheme != tc.scheme || link.Source.Kind != tc.source {
			t.Fatalf("%s => %+v %v", tc.raw, link, err)
		}
	}
}
func TestRejectMalformedAndUntrustedDeepLinks(t *testing.T) {
	for _, s := range []string{
		"",
		"https://makerworld.com/model",
		"bambustudio://open?file=http%3A%2F%2Fmakerworld.com%2Fmodel",
		"bambustudio://open?file=file%3A%2F%2Fetc%2Fpasswd",
		"bambustudio://open?file=https%3A%2F%2Fmakerworld.com.evil.test%2Ffile",
		"bambustudio://other?file=https%3A%2F%2Fmakerworld.com%2Ffile",
		"slicebridge://notimport?url=https%3A%2F%2Fmakerworld.com%2Ffile",
		"bambustudioopen://https%3A%2F%2F127.0.0.1%2Ffile",
		"bambustudioopen://https%3A%2F%2Fmakerworld.com%3A9999%2Ffile",
		"bambustudioopen://https%3A%2F%2Fmakerworld.com%40evil.test%2Ffile",
		"bambustudioopen://https%3A%2F%2Fevil.test%2Ffile",
		strings.Repeat("a", 40000),
	} {
		if link, err := ParseLink(s); err == nil {
			t.Errorf("incorrectly accepted %+v", link)
		}
	}
}
func TestMissingSourceNotMakerWorld(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "sample.3mf")
	if err := os.WriteFile(file, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}
	if out := Detect(file); out.Kind != "unknown" {
		t.Fatalf("pretended local 3mf is from MakerWorld: %+v", out)
	}
	if out := Detect(filepath.Join(tmp, "missing.3mf")); out.Kind != "unknown" {
		t.Fatalf("accepted missing file: %+v", out)
	}
	if out := Detect(filepath.Join(tmp, "sample.stl")); out.Kind != "unknown" {
		t.Fatalf("accepted non 3mf: %+v", out)
	}
	if runtime.GOOS == "windows" {
		// Test NTFS alternate data stream without touching browser associations.
		f, err := os.Create(file + ":Zone.Identifier")
		if err != nil {
			t.Skipf("Filesystem does not support alternate streams: %v", err)
		}
		_, err = f.WriteString("[ZoneTransfer]\r\nZoneId=3\r\nHostUrl=https://public-cdn.bblmw.com/a.3mf\r\nReferrerUrl=https://makerworld.com/en/models/999\r\n")
		if err != nil {
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
		if out := Detect(file); out.Kind != "makerworld" {
			t.Fatalf("NTFS origin not discovered: %+v", out)
		}
	}
}
