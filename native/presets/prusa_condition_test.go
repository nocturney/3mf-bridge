package presets

import "testing"

func TestPrusaConditionEvaluatesRealVendorForms(t *testing.T) {
	printer := prusaSection{
		"printer_model":   "X-Plus 4",
		"nozzle_diameter": "0.4;0.4",
	}
	cases := map[string]bool{
		`printer_model=="X-Plus 4"`:                             true,
		`printer_model=="Q1 Pro"`:                               false,
		`printer_model!="Q1 Pro"`:                               true,
		`printer_model=="X-Plus 4" and nozzle_diameter[0]==0.4`: true,
		`printer_model=="X-Plus 4" and nozzle_diameter[0]==0.2`: false,
		`nozzle_diameter[0]==0.40`:                              true,
		`nozzle_diameter[1]==0.4`:                               true,
		`nozzle_diameter[9]==""`:                                true,
		`printer_model=="Q1 Pro" or nozzle_diameter[0]==0.4`:    true,
		`(printer_model=="Q1 Pro" or printer_model=="X-Plus 4") and nozzle_diameter[0]==0.4`: true,
		`missing_setting==""`:                              true,
		`printer_model=~/^X-Plus 4$/`:                      true,
		`printer_model=~/PRINTER_VENDOR_PRUSA3D/`:          false,
		`printer_model!~/Q1/ and nozzle_diameter[0]==0.4`:  true,
		`printer_model=~/X\/Plus/`:                         false,
		`printer_model=~/.*X-Plus.*/ and num_extruders==1`: false,
	}
	for expr, want := range cases {
		got, err := evalPrusaCondition(expr, printer)
		if err != nil {
			t.Errorf("%s: unexpected error %v", expr, err)
			continue
		}
		if got != want {
			t.Errorf("%s = %v, want %v", expr, got, want)
		}
	}
}

func TestPrusaConditionFailsClosedOnUnsupportedSyntax(t *testing.T) {
	printer := prusaSection{"printer_model": "X-Plus 4"}
	for _, expr := range []string{
		``,
		`printer_model=~/X-Plus [`,
		`printer_model=~"X-Plus"`,
		`printer_model=~/unterminated`,
		`printer_model=~ "X"`,
		`printer_model!~/[a-z/`,
		`printer_model="X-Plus 4`,
		`printer_model=="X-Plus 4" and`,
		`printer_model == "X" trailing`,
		`printer_model=="X-Plus 4") `,
		`(printer_model=="X-Plus 4"`,
		`printer_model[99999]==""`,
		`printer_model>="X"`,
		`printer_model`,
		`printer_model=="X" or 1`,
		`system("ls")`,
	} {
		if _, err := evalPrusaCondition(expr, printer); err == nil {
			t.Errorf("%q accepted", expr)
		}
	}
	if _, err := evalPrusaCondition(`printer_model=="`+repeatByte('x', maxPrusaConditionBytes)+`"`, printer); err == nil {
		t.Error("oversized condition accepted")
	}
}

func repeatByte(c byte, n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = c
	}
	return string(out)
}
