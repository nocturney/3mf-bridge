package presets

import (
	"fmt"
	"strings"
	"testing"
)

func TestParsePrusaINIAcceptsRealProfileShapes(t *testing.T) {
	data := []byte("\xef\xbb\xbf# comment\r\n[vendor]\r\nname = QIDITechnology\r\n\r\n[print:*X-Plus 4*]\r\ninherits = *common*; *0.08mm*\r\nlayer_height = 0.2\r\n")
	ini, err := parsePrusaINI(data)
	if err != nil {
		t.Fatalf("valid profile rejected: %v", err)
	}
	if len(ini.order) != 2 || ini.order[0] != "vendor" || ini.order[1] != "print:*X-Plus 4*" {
		t.Fatalf("sections not kept in file order: %v", ini.order)
	}
	if got := ini.sections["vendor"]["name"]; got != "QIDITechnology" {
		t.Fatalf("vendor name = %q", got)
	}
	if got := ini.sections["print:*X-Plus 4*"]["inherits"]; got != "*common*; *0.08mm*" {
		t.Fatalf("inherits value not kept raw: %q", got)
	}
}

func TestParsePrusaINIRejectsMalformedInput(t *testing.T) {
	long := strings.Repeat("a", maxPrusaINILine+1)
	cases := map[string]string{
		"key outside section": "name = x\n",
		"no equals sign":      "[printer:a]\nnozzle 0.4\n",
		"unterminated header": "[printer:a\n",
		"empty section name":  "[]\n",
		"duplicate section":   "[printer:a]\n[printer:a]\n",
		"duplicate key":       "[printer:a]\nx = 1\nx = 2\n",
		"invalid key chars":   "[printer:a]\nBad-Key = 1\n",
		"NUL byte":            "[printer:a]\x00\n",
		"control character":   "[printer:a]\nx = a\x01b\n",
		"oversized line":      "[printer:a]\n" + long + "\n",
		"nested bracket name": "[printer:[a]]\n",
		"oversized value":     "[printer:a]\nx = " + strings.Repeat("v", maxPrusaValueBytes+1) + "\n",
	}
	for name, data := range cases {
		if _, err := parsePrusaINI([]byte(data)); err == nil {
			t.Errorf("%s: malformed profile accepted", name)
		}
	}
	if _, err := parsePrusaINI(make([]byte, maxPrusaINIBytes+1)); err == nil {
		t.Error("oversized file accepted")
	}
}

func TestParsePrusaINIBoundsSectionCount(t *testing.T) {
	var many strings.Builder
	for i := 0; i <= maxPrusaINISections; i++ {
		many.WriteString(fmt.Sprintf("[print:p%d]\n", i))
	}
	if _, err := parsePrusaINI([]byte(many.String())); err == nil {
		t.Error("section count above the limit accepted")
	}
}

func TestSplitPrusaList(t *testing.T) {
	cases := []struct {
		raw  string
		want []string
	}{
		{"", []string{}},
		{"0.4", []string{"0.4"}},
		{"*common*; *0.08mm*", []string{"*common*", "*0.08mm*"}},
		{`"Printer A";"Printer; B"`, []string{"Printer A", "Printer; B"}},
		{`"say \"hi\""`, []string{`say "hi"`}},
		{";;", []string{}},
	}
	for _, tc := range cases {
		got, err := splitPrusaList(tc.raw)
		if err != nil {
			t.Fatalf("splitPrusaList(%q): %v", tc.raw, err)
		}
		if strings.Join(got, "|") != strings.Join(tc.want, "|") || len(got) != len(tc.want) {
			t.Errorf("splitPrusaList(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
	if _, err := splitPrusaList(`"unterminated`); err == nil {
		t.Error("unterminated quote accepted")
	}
}
