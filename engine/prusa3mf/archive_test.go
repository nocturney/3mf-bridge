package prusa3mf

import "testing"

func TestCheckEntryNameAcceptsPackageParts(t *testing.T) {
	for _, name := range []string{
		"3D/3dmodel.model",
		"Metadata/plate_1.json",
		"[Content_Types].xml",
		"Metadata/",
	} {
		if err := checkEntryName(name); err != nil {
			t.Fatalf("checkEntryName(%q) rejected a safe name: %v", name, err)
		}
	}
}

func TestCheckEntryNameRejectsUnsafeNames(t *testing.T) {
	for _, name := range []string{
		"",
		"/",
		"/absolute.txt",
		"../escape.txt",
		"a/../b.txt",
		"a/./b.txt",
		"a//b.txt",
		`a\b.txt`,
		"C:/windows.txt",
		"control\x00byte.txt",
	} {
		if err := checkEntryName(name); ErrorCode(err) != "UNSAFE_ENTRY" {
			t.Fatalf("checkEntryName(%q) code = %s (%v), want UNSAFE_ENTRY", name, ErrorCode(err), err)
		}
	}
}
