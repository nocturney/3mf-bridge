package prusa3mf

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"sort"
	"strings"
)

// Resource limits. They bound memory and work for hostile or accidental input
// before any XML is parsed. Nothing is ever extracted to disk.
const (
	maxSourceBytes      = 250 << 20
	maxEntries          = 512
	maxModelBytes       = 256 << 20
	maxConfigBytes      = 8 << 20
	maxTotalBytes       = 512 << 20
	ratioCheckFloor     = 1 << 20
	maxCompressionRatio = 100
	maxEntryNameBytes   = 512
)

// archive is a validated view of a source ZIP. Only regular, safely named,
// size-bounded entries are kept; directories are skipped.
type archive struct {
	zr    *zip.ReadCloser
	files map[string]*zip.File
}

func openArchive(path string) (*archive, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, fail("SOURCE_INVALID", "%v", err)
	}
	if !st.Mode().IsRegular() || st.Size() == 0 || st.Size() > maxSourceBytes {
		return nil, fail("SOURCE_INVALID", "3MF must be a nonempty regular file up to 250 MB")
	}
	if err := checkEntryCount(path); err != nil {
		return nil, err
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fail("SOURCE_INVALID", "not a readable 3MF archive: %v", err)
	}
	a := &archive{zr: zr, files: make(map[string]*zip.File, len(zr.File))}
	if err := a.validate(); err != nil {
		zr.Close()
		return nil, err
	}
	return a, nil
}

// checkEntryCount reads the End of Central Directory record before archive/zip
// builds its entry table, so a hostile entry count is refused before allocation.
func checkEntryCount(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fail("SOURCE_INVALID", "%v", err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return fail("SOURCE_INVALID", "%v", err)
	}
	// The record is 22 bytes followed by a comment of at most 65535 bytes.
	tailLen := min(st.Size(), 22+65535)
	tail := make([]byte, tailLen)
	if _, err := io.ReadFull(io.NewSectionReader(f, st.Size()-tailLen, tailLen), tail); err != nil {
		return fail("SOURCE_INVALID", "cannot read ZIP directory: %v", err)
	}
	idx := bytes.LastIndex(tail, []byte{0x50, 0x4b, 0x05, 0x06})
	if idx < 0 || len(tail)-idx < 22 {
		return fail("SOURCE_INVALID", "not a ZIP archive")
	}
	if n := binary.LittleEndian.Uint16(tail[idx+10:]); int(n) > maxEntries {
		return fail("ARCHIVE_LIMIT", "archive contains %d entries (limit %d)", n, maxEntries)
	}
	return nil
}

func (a *archive) validate() error {
	if len(a.zr.File) > maxEntries {
		return fail("ARCHIVE_LIMIT", "archive contains %d entries (limit %d)", len(a.zr.File), maxEntries)
	}
	seen := make(map[string]bool, len(a.zr.File))
	var total uint64
	for _, f := range a.zr.File {
		if err := checkEntryName(f.Name); err != nil {
			return err
		}
		key := strings.ToLower(strings.TrimSuffix(f.Name, "/"))
		if seen[key] {
			return fail("UNSAFE_ENTRY", "duplicate archive entry %q", f.Name)
		}
		seen[key] = true
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		size := f.UncompressedSize64
		if size > maxModelBytes || total+size > maxTotalBytes {
			return fail("ARCHIVE_LIMIT", "archive exceeds the decompressed size limit at %q", f.Name)
		}
		total += size
		if size > ratioCheckFloor && (f.CompressedSize64 == 0 || size/f.CompressedSize64 > maxCompressionRatio) {
			return fail("ZIP_BOMB", "entry %q expands beyond the allowed compression ratio", f.Name)
		}
		a.files[f.Name] = f
	}
	return nil
}

// checkEntryName refuses absolute paths, parent references, backslashes,
// drive-style colons, control characters and empty segments.
func checkEntryName(name string) error {
	clean := strings.TrimSuffix(name, "/")
	if clean == "" || len(name) > maxEntryNameBytes {
		return fail("UNSAFE_ENTRY", "unsafe archive entry %q", name)
	}
	for i := 0; i < len(name); i++ {
		if name[i] < 0x20 || name[i] == '\\' || name[i] == ':' {
			return fail("UNSAFE_ENTRY", "unsafe archive entry %q", name)
		}
	}
	for _, seg := range strings.Split(clean, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return fail("UNSAFE_ENTRY", "unsafe archive entry %q", name)
		}
	}
	return nil
}

func (a *archive) close() {
	a.zr.Close()
}

func (a *archive) has(name string) bool {
	_, ok := a.files[name]
	return ok
}

func (a *archive) names() []string {
	out := make([]string, 0, len(a.files))
	for name := range a.files {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// read returns one entry, refusing content larger than limit even when the
// declared size is smaller (declared sizes can be forged).
func (a *archive) read(name string, limit int64) ([]byte, error) {
	f, ok := a.files[name]
	if !ok {
		return nil, fail("SOURCE_INVALID", "package is missing %s", name)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, fail("SOURCE_INVALID", "cannot open %s: %v", name, err)
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err != nil {
		return nil, fail("SOURCE_INVALID", "cannot read %s: %v", name, err)
	}
	if int64(len(data)) > limit {
		return nil, fail("ARCHIVE_LIMIT", "%s exceeds %d bytes", name, limit)
	}
	return data, nil
}
