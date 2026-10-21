package prusa3mf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// rawModel is a one-object model whose triangle elements are supplied verbatim.
func rawModel(triangles string) string {
	return `<model unit="millimeter" xmlns="` + coreNS + `"><resources>` +
		`<object id="1" type="model"><mesh>` + tetraVertices + `<triangles>` + triangles + `</triangles></mesh></object>` +
		`</resources><build><item objectid="1"/></build></model>`
}

// TestRefusedPackagesFailWithStableCodes covers every unsupported or unsafe
// case. Each one must fail with its documented code and leave no output behind.
func TestRefusedPackagesFailWithStableCodes(t *testing.T) {
	cases := []struct {
		name string
		want string
		edit func(f map[string][]byte)
	}{
		{"paint split tree", "UNSUPPORTED_PAINT", func(f map[string][]byte) {
			f[modelPart] = []byte(tetraModel(defaultBuild, "1"))
		}},
		{"paint channel beyond table", "CHANNEL_NOT_IN_SOURCE", func(f map[string][]byte) {
			f[modelPart] = []byte(tetraModel(defaultBuild, "C"))
		}},
		{"paint on placeholder gap", "CHANNEL_NOT_IN_SOURCE", func(f map[string][]byte) {
			f[sliceInfoPart] = []byte(sliceInfo(filament(1, "PLA", "#FF0000"), filament(3, "PLA", "#0000FF")))
			f[modelPart] = []byte(tetraModel(defaultBuild, "8"))
		}},
		{"object extruder beyond table", "CHANNEL_NOT_IN_SOURCE", func(f map[string][]byte) {
			f[modelSettingsPart] = []byte(`<config><object id="1"><metadata key="extruder" value="9"/></object></config>`)
		}},
		{"doctype with entity", "UNSUPPORTED_XML", func(f map[string][]byte) {
			f[modelPart] = []byte(`<!DOCTYPE model [<!ENTITY e "x">]><model xmlns="` + coreNS + `"><resources/><build/></model>`)
		}},
		{"components", "UNSUPPORTED_COMPONENTS", func(f map[string][]byte) {
			f[modelPart] = []byte(`<model xmlns="` + coreNS + `"><resources><object id="1" type="model"><components><component objectid="2"/></components></object></resources><build><item objectid="1"/></build></model>`)
		}},
		{"unplaced object", "UNSUPPORTED_UNREFERENCED_OBJECT", func(f map[string][]byte) {
			f[modelPart] = []byte(`<model xmlns="` + coreNS + `"><resources>` + tetraObject(1) + tetraObject(2) + `</resources><build><item objectid="1"/></build></model>`)
		}},
		{"two parts", "UNSUPPORTED_MULTIPART", func(f map[string][]byte) {
			f[modelSettingsPart] = []byte(`<config><object id="1"><part id="2"/><part id="3"/></object></config>`)
		}},
		{"part triangle ranges", "UNSUPPORTED_MULTIPART", func(f map[string][]byte) {
			f[modelSettingsPart] = []byte(`<config><object id="1"><part id="2" subtype="normal_part"><metadata key="firstid" value="0"/><metadata key="lastid" value="3"/></part></object></config>`)
		}},
		{"modifier part", "UNSUPPORTED_MODIFIER_PART", func(f map[string][]byte) {
			f[modelSettingsPart] = []byte(`<config><object id="1"><part id="2" subtype="support_blocker"/></object></config>`)
		}},
		{"unknown triangle attribute", "UNSUPPORTED_ATTRIBUTE", func(f map[string][]byte) {
			f[modelPart] = []byte(rawModel(`<triangle v1="0" v2="1" v3="2" seam="1"/>`))
		}},
		{"source already carries prusa paint", "UNSUPPORTED_ATTRIBUTE", func(f map[string][]byte) {
			f[modelPart] = []byte(rawModel(`<triangle xmlns:slic3rpe="` + slic3rpeNS + `" v1="0" v2="1" v3="2" slic3rpe:mmu_segmentation="4"/>`))
		}},
		{"triangle outside mesh", "SOURCE_INVALID", func(f map[string][]byte) {
			f[modelPart] = []byte(rawModel(`<triangle v1="0" v2="1" v3="9"/>`))
		}},
		{"non-finite vertex", "SOURCE_INVALID", func(f map[string][]byte) {
			f[modelPart] = []byte(strings.Replace(tetraModel(defaultBuild), `x="10"`, `x="NaN"`, 1))
		}},
		{"unsupported unit", "UNSUPPORTED_UNIT", func(f map[string][]byte) {
			f[modelPart] = []byte(strings.Replace(tetraModel(defaultBuild), `unit="millimeter"`, `unit="furlong"`, 1))
		}},
		{"support object type", "UNSUPPORTED_OBJECT_TYPE", func(f map[string][]byte) {
			f[modelPart] = []byte(strings.Replace(tetraModel(defaultBuild), `type="model"`, `type="support"`, 1))
		}},
		{"invalid filament colour", "CHANNEL_INVALID", func(f map[string][]byte) {
			f[sliceInfoPart] = []byte(sliceInfo(filament(1, "PLA", "red"), filament(2, "PLA", "#00FF00")))
		}},
		{"same filament id with different settings", "FILAMENT_TABLE_CONFLICT", func(f map[string][]byte) {
			f[sliceInfoPart] = []byte(sliceInfo(filament(1, "PLA", "#FF0000"), filament(1, "PLA", "#00FF00")))
		}},
		{"slice info disagrees with project colours", "FILAMENT_TABLE_CONFLICT", func(f map[string][]byte) {
			f[projectSettingsPart] = []byte(`{"filament_colour":["#00FF00","#00FF00"],"filament_type":["PLA","PLA"]}`)
		}},
		{"no filament table", "NO_FILAMENTS", func(f map[string][]byte) {
			delete(f, sliceInfoPart)
		}},
		{"unsafe parent path", "UNSAFE_ENTRY", func(f map[string][]byte) {
			f["../evil.txt"] = []byte("x")
		}},
		{"absolute path", "UNSAFE_ENTRY", func(f map[string][]byte) {
			f["/abs.txt"] = []byte("x")
		}},
		{"backslash path", "UNSAFE_ENTRY", func(f map[string][]byte) {
			f[`Metadata\x.txt`] = []byte("x")
		}},
		{"empty segment", "UNSAFE_ENTRY", func(f map[string][]byte) {
			f["Metadata//x.txt"] = []byte("x")
		}},
		{"case-only duplicate entry", "UNSAFE_ENTRY", func(f map[string][]byte) {
			f["METADATA/model_settings.config"] = []byte("x")
		}},
		{"extra model part", "UNSUPPORTED_MULTI_MODEL", func(f map[string][]byte) {
			f["3D/other.model"] = []byte(tetraModel(defaultBuild))
		}},
		{"zip bomb", "ZIP_BOMB", func(f map[string][]byte) {
			f["Metadata/junk.txt"] = make([]byte, 4<<20)
		}},
		{"too many entries", "ARCHIVE_LIMIT", func(f map[string][]byte) {
			for i := 0; i < 600; i++ {
				f[fmt.Sprintf("Metadata/f%03d.txt", i)] = []byte("x")
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := basePackage()
			tc.edit(files)
			dst := filepath.Join(t.TempDir(), "out.3mf")
			_, err := Convert(writePackage(t, files), dst)
			if got := ErrorCode(err); got != tc.want {
				t.Fatalf("code = %s (%v), want %s", got, err, tc.want)
			}
			if _, statErr := os.Stat(dst); !os.IsNotExist(statErr) {
				t.Fatal("refused conversion left an output file behind")
			}
		})
	}
}

func TestRefusesNonZIPSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plain.3mf")
	if err := os.WriteFile(path, []byte("this is plain text, not a 3MF package"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Convert(path, filepath.Join(t.TempDir(), "out.3mf"))
	if ErrorCode(err) != "SOURCE_INVALID" {
		t.Fatalf("code = %s (%v), want SOURCE_INVALID", ErrorCode(err), err)
	}
}

// TestVerifierRejectsChangedProject checks that publication is gated on the
// output matching the source project, not merely on the output parsing.
func TestVerifierRejectsChangedProject(t *testing.T) {
	p, err := readSource(copyFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	data, err := encodeOutput(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyOutput(data, p); err != nil {
		t.Fatalf("verifier rejected the matching project: %v", err)
	}
	p.items[0].transform = &[12]float64{1, 0, 0, 0, 1, 0, 0, 0, 1, 5, 0, 0}
	if err := verifyOutput(data, p); err == nil {
		t.Fatal("verifier accepted a changed build transform")
	}
}
