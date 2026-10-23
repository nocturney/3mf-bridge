package bridge

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const qaOPCNS = "http://schemas.openxmlformats.org/package/2006/relationships"

func TestCacheReferenceRewriteProducesValidNamespacedOPC(t *testing.T) {
	input := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?><Relationships xmlns=\"" + qaOPCNS + "\">" +
		"<Relationship Id=\"model\" Target=\"/3D/3dmodel.model\" Type=\"model\"/>" +
		"<Relationship Id=\"preview\" Target=\"/Metadata/plate_1.png\" Type=\"thumbnail\"/>" +
		"<Relationship Id=\"cache\" Target=\"/Metadata/plate_1.gcode\" Type=\"cache\"/>" +
		"</Relationships>")
	out, err := stripCacheReferences(input)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		XMLName xml.Name
		Items   []struct {
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	if err := xml.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("relationships XML is invalid: %v; content %s", err, out)
	}
	if parsed.XMLName.Space != qaOPCNS {
		t.Fatalf("namespace lost: %s", parsed.XMLName.Space)
	}
	if len(parsed.Items) != 2 {
		t.Fatalf("want model + thumbnail, got %+v", parsed.Items)
	}
	if !bytes.Contains(out, []byte("plate_1.png")) || bytes.Contains(out, []byte("plate_1.gcode")) {
		t.Fatalf("valid thumbnail removed or stale cache retained: %s", out)
	}
}

func TestStripCachePreservesNamespacedThumbnailRelationshipBytes(t *testing.T) {
	relationship := []byte("<?xml version=\"1.0\"?><Relationships xmlns=\"" + qaOPCNS + "\">" +
		"<Relationship Target=\"/Metadata/plate_1.png\" Type=\"thumbnail\" Id=\"one\"/></Relationships>")
	src := filepath.Join(t.TempDir(), "source.3mf")
	dst := filepath.Join(t.TempDir(), "target.3mf")
	f, err := os.Create(src)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("_rels/.rels")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Write(relationship); err != nil {
		t.Fatal(err)
	}
	if err = zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	removed, err := stripSliceCache(src, dst, nil)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("unexpected cache count: %d", removed)
	}
	r, err := zip.OpenReader(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if len(r.File) != 1 {
		t.Fatalf("unexpected output file count: %d", len(r.File))
	}
	rc, err := r.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, relationship) {
		t.Fatalf("thumbnail-only OPC XML must be preserved byte-for-byte, got %q", out)
	}
}

func TestCacheReferenceFilterIgnoresThumbnailAndRejectsStaleCache(t *testing.T) {
	for _, tc := range []struct {
		data   string
		expect bool
	}{
		{"Target=/Metadata/plate_1.png", false},
		{"Target=/Metadata/plate_1_small.png", false},
		{"Target=/Metadata/plate_1.gcode", true},
		{"Target=/Metadata/plate_1.gcode.md5", true},
		{"PartName=/Metadata/plate_2.json", true},
		{"<Relationship Target='plate_123.GCODE'/>", true},
	} {
		if got := cacheReference.MatchString(tc.data); got != tc.expect {
			t.Errorf("%q: got=%v want=%v", tc.data, got, tc.expect)
		}
	}
	if !strings.Contains(qaOPCNS, "relationships") {
		t.Fatal("namespace fixture malformed")
	}
}
