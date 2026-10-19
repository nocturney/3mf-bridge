package main

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nocturney/3mf-bridge/native/makerworld"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (t transportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return t(req) }
func projectBytes(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for _, entry := range []string{"[Content_Types].xml", "3D/3dmodel.model"} {
		f, err := w.Create(entry)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.Write([]byte("<model/>")); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestTrustedDownloadPersistsValid3MF(t *testing.T) {
	source := "https://makerworld.com/api/model.3mf?signed=secret"
	link, err := makerworld.ParseLink("slicebridge://import?url=" + url.QueryEscape(source))
	if err != nil {
		t.Fatal(err)
	}
	payload := projectBytes(t)
	client := &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != source {
			t.Fatalf("unexpected origin")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(payload)), Request: req, ContentLength: int64(len(payload))}, nil
	})}
	dir := t.TempDir()
	for i := 0; i < 2; i++ {
		file, err := download3MF(context.Background(), client, link, dir)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Dir(file.Path) != dir || !strings.HasSuffix(file.Name, ".3mf") {
			t.Fatalf("bad output %+v", file)
		}
		if err = verify3MFArchive(file.Path); err != nil {
			t.Fatal(err)
		}
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 distinct files, got %v", files)
	}
}
func TestDownloadRejectsFailedAuthAndHTML(t *testing.T) {
	src := "https://public-cdn.bblmw.com/a.3mf"
	link, err := makerworld.ParseLink("slicebridge://import?url=" + url.QueryEscape(src))
	if err != nil {
		t.Fatal(err)
	}
	for _, response := range []struct {
		status  int
		payload []byte
	}{
		{http.StatusForbidden, []byte("forbidden")}, {http.StatusOK, []byte("<html>not a model</html>")},
	} {
		client := &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: response.status, Body: io.NopCloser(bytes.NewReader(response.payload)), Request: req, ContentLength: int64(len(response.payload))}, nil
		})}
		folder := t.TempDir()
		if out, err := download3MF(context.Background(), client, link, folder); err == nil {
			t.Fatalf("invalid import accepted: %+v", out)
		}
		items, err := os.ReadDir(folder)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 0 {
			t.Fatalf("partial or unsafe output left behind")
		}
	}
}
func TestURLSanitizerRejectsUntrustedDestinations(t *testing.T) {
	for _, raw := range []string{
		"http://makerworld.com/a", "https://evil.test/a", "https://makerworld.com.evil.test/a",
		"https://127.0.0.1/a", "https://makerworld.com:9999/a", "file:///etc/passwd",
	} {
		if trustedHTTPS(raw) {
			t.Errorf("trusted bad redirect %q", raw)
		}
	}
	for _, raw := range []string{"https://makerworld.com/a", "https://public-cdn.bblmw.com/a.3mf"} {
		if !trustedHTTPS(raw) {
			t.Errorf("rejected expected origin %q", raw)
		}
	}
}
