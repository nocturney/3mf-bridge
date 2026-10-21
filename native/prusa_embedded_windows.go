package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	prusaBundleMagic       = "SLICEBRIDGE_PRUSA_V1"
	prusaBundleMaxSize     = 128 << 20
	prusaBundleTrailerSize = len(prusaBundleMagic) + 8 + sha256.Size
)

// locateEmbeddedPrusaEngine finds a verified, appended ZIP in our own Windows
// portable executable. No Python, installer or external sidecar is required.
// The engine is extracted only to the current user's versioned cache directory.
func locateEmbeddedPrusaEngine(self string) (string, bool, error) {
	f, err := os.Open(self)
	if err != nil {
		return "", false, err
	}
	defer f.Close()
	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return "", false, err
	}
	if size < int64(prusaBundleTrailerSize) {
		return "", false, nil
	}
	trailer := make([]byte, prusaBundleTrailerSize)
	if _, err := f.ReadAt(trailer, size-int64(len(trailer))); err != nil {
		return "", false, err
	}
	if !bytes.Equal(trailer[:len(prusaBundleMagic)], []byte(prusaBundleMagic)) {
		return "", false, nil // Older developer builds may use a separate sidecar.
	}
	n := binary.LittleEndian.Uint64(trailer[len(prusaBundleMagic):])
	if n == 0 || n > prusaBundleMaxSize || n > uint64(size-int64(prusaBundleTrailerSize)) {
		return "", true, errors.New("invalid embedded engine length")
	}
	start := size - int64(prusaBundleTrailerSize) - int64(n)
	expected := trailer[len(prusaBundleMagic)+8:]
	hash := sha256.New()
	if _, err := io.Copy(hash, io.NewSectionReader(f, start, int64(n))); err != nil {
		return "", true, err
	}
	if !bytes.Equal(hash.Sum(nil), expected) {
		return "", true, errors.New("embedded engine checksum mismatch")
	}
	zr, err := zip.NewReader(io.NewSectionReader(f, start, int64(n)), int64(n))
	if err != nil {
		return "", true, err
	}
	if len(zr.File) != 2 {
		return "", true, errors.New("embedded engine must contain exactly two files")
	}
	files := make(map[string][]byte, 2)
	for _, entry := range zr.File {
		limit := int64(1 << 20)
		switch entry.Name {
		case "slicebridge-prusa-engine.exe":
			limit = 96 << 20
		case "panda2prusa-LICENSE.txt":
		default:
			return "", true, fmt.Errorf("unexpected embedded file %q", entry.Name)
		}
		if _, present := files[entry.Name]; present || entry.FileInfo().IsDir() || entry.UncompressedSize64 > uint64(limit) {
			return "", true, errors.New("duplicate, directory or oversized embedded file")
		}
		reader, err := entry.Open()
		if err != nil {
			return "", true, err
		}
		data, readErr := io.ReadAll(io.LimitReader(reader, limit+1))
		closeErr := reader.Close()
		if readErr != nil {
			return "", true, readErr
		}
		if closeErr != nil {
			return "", true, closeErr
		}
		if int64(len(data)) > limit || uint64(len(data)) != entry.UncompressedSize64 {
			return "", true, errors.New("invalid embedded file size")
		}
		files[entry.Name] = data
	}
	engine := files["slicebridge-prusa-engine.exe"]
	license := files["panda2prusa-LICENSE.txt"]
	if len(engine) < 2 || string(engine[:2]) != "MZ" ||
		!bytes.Contains(license, []byte("MIT License")) ||
		!bytes.Contains(license, []byte("Permission is hereby granted")) {
		return "", true, errors.New("embedded engine or upstream license invalid")
	}
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return "", true, err
	}
	dir := filepath.Join(cacheRoot, "SliceBridge", "PrusaEngine", fmt.Sprintf("%x", expected[:12]))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", true, err
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", true, errors.New("unsafe embedded engine cache directory")
	}
	if err := storeVerifiedPrusaCacheFile(dir, "panda2prusa-LICENSE.txt", license, 0o600); err != nil {
		return "", true, err
	}
	if err := storeVerifiedPrusaCacheFile(dir, "slicebridge-prusa-engine.exe", engine, 0o700); err != nil {
		return "", true, err
	}
	return filepath.Join(dir, "slicebridge-prusa-engine.exe"), true, nil
}

// Atomic, hash-checked extraction. A tampered cached engine is never trusted.
// A concurrent SliceBridge instance may have completed the same write first.
func storeVerifiedPrusaCacheFile(dir, name string, data []byte, mode os.FileMode) error {
	dest := filepath.Join(dir, name)
	want := sha256.Sum256(data)
	isValid := func() bool {
		info, err := os.Lstat(dest)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
			info.Size() != int64(len(data)) {
			return false
		}
		f, err := os.Open(dest)
		if err != nil {
			return false
		}
		defer f.Close()
		h := sha256.New()
		_, err = io.Copy(h, f)
		return err == nil && bytes.Equal(h.Sum(nil), want[:])
	}
	if isValid() {
		return nil
	}
	if info, err := os.Lstat(dest); err == nil {
		if !info.Mode().IsRegular() {
			return errors.New("refusing to replace unsafe cached Prusa engine file")
		}
		if err := os.Remove(dest); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".engine-staging-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), dest); err != nil && !isValid() {
		return err
	}
	if !isValid() {
		return errors.New("cached Prusa engine failed post-extraction hash check")
	}
	return nil
}
