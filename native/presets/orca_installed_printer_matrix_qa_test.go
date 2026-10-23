package presets

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/nocturney/slicebridge/engine/bridge"
)

// Explicit local QA; skipped by normal CI. Never updates a saved slicer profile
// or initiates a print. Produces an auditable report per installed application,
// printer model and physical nozzle.
type installedMatrixRow struct {
	Slicer       string `json:"slicer"`
	Vendor       string `json:"vendor"`
	Model        string `json:"model"`
	Machine      string `json:"machine"`
	Nozzle       string `json:"nozzle,omitempty"`
	TargetID     string `json:"targetId,omitempty"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
	NativeSliced bool   `json:"nativeSliced"`
	Selectable   bool   `json:"selectable"`
}
type installedMatrixReport struct {
	GeneratedAt string                    `json:"generatedAt"`
	Scope       string                    `json:"scope"`
	Locations   []string                  `json:"locations"`
	Rows        []installedMatrixRow      `json:"rows"`
	AppCounts   map[string]map[string]int `json:"appCounts"`
	Count       map[string]int            `json:"count"`
}

func TestQAInstalledOrcaPrinterMatrix(t *testing.T) {
	if os.Getenv("QA_INSTALLED_ORCA_MATRIX") != "1" {
		t.Skip("local opt-in QA only")
	}
	reportDir := os.Getenv("QA_INSTALLED_ORCA_REPORT")
	if reportDir == "" {
		t.Fatal("QA_INSTALLED_ORCA_REPORT required")
	}
	if err := os.MkdirAll(reportDir, 0700); err != nil {
		t.Fatal(err)
	}
	convert := os.Getenv("QA_INSTALLED_ORCA_CONVERT") == "1"
	source := filepath.Join("..", "..", "tests", "fixtures", "bambu-one-color.3mf")
	if _, err := os.Stat(source); err != nil {
		t.Fatal(err)
	}
	inputGeometry, err := qaMatrixModels(source)
	if err != nil {
		t.Fatal(err)
	}
	locations := installedOrcaLocations()
	report := installedMatrixReport{GeneratedAt: time.Now().UTC().Format(time.RFC3339), Scope: "locally installed Orca-family apps; no hardware and no slicer profile modifications", Rows: []installedMatrixRow{}, Locations: []string{}, Count: map[string]int{}, AppCounts: map[string]map[string]int{}}
	if len(locations) == 0 {
		t.Fatal("no installed Orca locations were found")
	}
	selectable, _ := scanOrca(locations)
	selectableIDs := make(map[string]bool, len(selectable))
	for _, c := range selectable {
		selectableIDs[c.target.ID] = true
	}
	for _, loc := range locations {
		report.Locations = append(report.Locations, loc.name)
		if report.AppCounts[loc.name] == nil {
			report.AppCounts[loc.name] = map[string]int{}
		}
		vendors, err := os.ReadDir(loc.root)
		if err != nil {
			t.Log("missing root", loc.name, err)
			continue
		}
		for _, v := range vendors {
			if !v.IsDir() || v.Name() == "BBL" || v.Name() == "OrcaFilamentLibrary" || !orcaVendorOwnedByApp(loc.id, v.Name()) {
				continue
			}
			candidates, rejects := scanOrcaVendor(loc, v.Name())
			for _, rej := range rejects {
				row := installedMatrixRow{Slicer: loc.name, Vendor: v.Name(), Model: rej.Machine, Machine: rej.Machine, Status: "REJECTED_PROFILE", Reason: rej.Reason}
				report.Rows = append(report.Rows, row)
				report.Count[row.Status]++
				report.AppCounts[loc.name][row.Status]++
			}
			for _, c := range candidates {
				row := installedMatrixRow{Slicer: loc.name, Vendor: v.Name(), Model: c.target.Model, Machine: c.target.Machine, Nozzle: c.target.Nozzle, TargetID: c.target.ID, Status: "PROFILE_RESOLVED", Selectable: selectableIDs[c.target.ID]}
				if !row.Selectable {
					row.Status = "FALLBACK_NOT_OFFERED"
					row.Reason = "duplicate or nonpreferred target in another Orca application"
				}
				if convert && (os.Getenv("QA_MATRIX_FILTER_MACHINE") == "" || strings.Contains(c.target.Machine, os.Getenv("QA_MATRIX_FILTER_MACHINE"))) {
					b, err := json.Marshal(c.bundle)
					if err != nil {
						t.Fatal(err)
					}
					filename := filepath.Join(reportDir, fmt.Sprintf("target-%s-%03d.json", qaMatrixSlug(loc.id), len(report.Rows)))
					if err = os.WriteFile(filename, b, 0600); err != nil {
						t.Fatal(err)
					}
					out := filepath.Join(reportDir, fmt.Sprintf("output-%s-%03d.3mf", qaMatrixSlug(loc.id), len(report.Rows)))
					_ = os.Remove(out) // only isolated QA outputs
					_, err = bridge.ConvertWithOrcaBundle(source, out, filename)
					if err != nil {
						row.Status = "CONVERSION_FAILED"
						row.Reason = err.Error()
					} else if err = qaMatrixValidate3MF(out, c, inputGeometry); err != nil {
						row.Status = "VALIDATION_FAILED"
						row.Reason = err.Error()
					} else {
						row.Status = "CONVERTED_3MF_VALID"
					}
					_ = os.Remove(filename)
					if os.Getenv("QA_MATRIX_KEEP_OUTPUT") != "1" {
						_ = os.Remove(out)
					}
				}
				report.Rows = append(report.Rows, row)
				report.Count[row.Status]++
				report.AppCounts[loc.name][row.Status]++
			}
		}
	}
	sort.Slice(report.Rows, func(i, j int) bool {
		a, b := report.Rows[i], report.Rows[j]
		return a.Slicer+"|"+a.Vendor+"|"+a.Machine+"|"+a.Status < b.Slicer+"|"+b.Vendor+"|"+b.Machine+"|"+b.Status
	})
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(reportDir, "installed-orca-printer-matrix.json")
	if err = os.WriteFile(dest, data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("ACTUAL_LOCAL_INSTALLED_SLICER_MATRIX locations=%v totals=%v perApp=%v report=%s", report.Locations, report.Count, report.AppCounts, dest)
}

func qaMatrixSlug(s string) string {
	return strings.NewReplacer("/", "-", " ", "-", "\\", "-").Replace(s)
}

// Fingerprint geometric semantics, not XML quote style. The converter may
// reserialize 3MF using double quotes even when the source used single quotes.
func qaMatrixModels(path string) (map[string]string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	result := map[string]string{}
	for _, e := range r.File {
		if !strings.HasSuffix(strings.ToLower(e.Name), ".model") {
			continue
		}
		if e.UncompressedSize64 > 80<<20 {
			return nil, fmt.Errorf("oversized geometry")
		}
		f, err := e.Open()
		if err != nil {
			return nil, err
		}
		dec := xml.NewDecoder(io.LimitReader(f, 80<<20))
		h := sha256.New()
		nodes := 0
		for {
			token, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				f.Close()
				return nil, fmt.Errorf("model XML invalid: %w", err)
			}
			start, ok := token.(xml.StartElement)
			if !ok {
				continue
			}
			switch start.Name.Local {
			case "object", "mesh", "vertex", "triangle", "component", "item":
				attrs := make([]string, 0, len(start.Attr))
				for _, a := range start.Attr {
					if a.Name.Local == "xmlns" {
						continue
					}
					attrs = append(attrs, a.Name.Local+"="+a.Value)
				}
				sort.Strings(attrs)
				fmt.Fprintf(h, "%s:%s\n", start.Name.Local, strings.Join(attrs, "|"))
				nodes++
			}
		}
		f.Close()
		if nodes == 0 {
			return nil, fmt.Errorf("model lacks geometry nodes")
		}
		result[e.Name] = hex.EncodeToString(h.Sum(nil))
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no input 3D models")
	}
	return result, nil
}
func qaMatrixValidate3MF(path string, c orcaCandidate, inputGeometry map[string]string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()
	var cfg map[string]any
	var scripts, checked bool
	for _, e := range r.File {
		if e.Name == "Metadata/project_settings.config" {
			f, err := e.Open()
			if err != nil {
				return err
			}
			err = json.NewDecoder(io.LimitReader(f, 8<<20)).Decode(&cfg)
			f.Close()
			if err != nil {
				return err
			}
		}
		if e.Name == "_rels/.rels" {
			f, err := e.Open()
			if err != nil {
				return err
			}
			var value struct{ XMLName xml.Name }
			err = xml.NewDecoder(io.LimitReader(f, 2<<20)).Decode(&value)
			f.Close()
			if err != nil {
				return fmt.Errorf("invalid relationships XML: %w", err)
			}
		}
	}
	if cfg == nil {
		return fmt.Errorf("missing project config")
	}
	if cfg["printer_model"] != c.target.Model {
		return fmt.Errorf("wrong target model=%v", cfg["printer_model"])
	}
	nozzles, ok := cfg["nozzle_diameter"].([]any)
	if !ok || len(nozzles) != c.target.PhysicalTools {
		return fmt.Errorf("physical nozzle array invalid: %v", cfg["nozzle_diameter"])
	}
	for _, n := range nozzles {
		if fmt.Sprint(n) != c.target.Nozzle {
			return fmt.Errorf("nozzle %v != approved %s", n, c.target.Nozzle)
		}
	}
	start, _ := cfg["machine_start_gcode"].(string)
	end, _ := cfg["machine_end_gcode"].(string)
	if start == "" || end == "" {
		return fmt.Errorf("vendor machine scripts missing")
	}
	if strings.Contains(start, "UNSAFE_BAMBU_SCRIPT") || strings.Contains(end, "UNSAFE_BAMBU_SCRIPT") {
		return fmt.Errorf("Bambu script leaked")
	}
	scripts = true
	outputGeometry, err := qaMatrixModels(path)
	if err != nil {
		return err
	}
	if len(inputGeometry) != len(outputGeometry) {
		return fmt.Errorf("model count changed input=%d output=%d", len(inputGeometry), len(outputGeometry))
	}
	for name, h := range inputGeometry {
		if outputGeometry[name] != h {
			return fmt.Errorf("geometry changed %s", name)
		}
	}
	checked = true
	if !scripts || !checked {
		return fmt.Errorf("mandatory validation skipped")
	}
	return nil
}
