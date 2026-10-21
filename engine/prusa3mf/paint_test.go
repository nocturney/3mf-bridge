package prusa3mf

import "testing"

// TestDecodePaintRepositoryFixtureCodes pins the two codes that the repository
// fixture uses: "4" is channel 1 and "2C" is channel 5.
func TestDecodePaintRepositoryFixtureCodes(t *testing.T) {
	cases := map[string]int{
		"":   0,
		"0":  0,
		"4":  1,
		"8":  2,
		"C":  3,
		"1C": 4,
		"2C": 5,
		"2c": 5,
		"FC": 18,
	}
	for code, want := range cases {
		got, err := decodePaint(code)
		if err != nil {
			t.Fatalf("decodePaint(%q) error: %v", code, err)
		}
		if got != want {
			t.Fatalf("decodePaint(%q) = %d, want %d", code, got, want)
		}
	}
}

func TestPaintCodecRoundTrip(t *testing.T) {
	for channel := 1; channel <= maxPaintChannel; channel++ {
		code, err := encodePaint(channel)
		if err != nil {
			t.Fatalf("encodePaint(%d) error: %v", channel, err)
		}
		got, err := decodePaint(code)
		if err != nil || got != channel {
			t.Fatalf("round trip of channel %d via %q gave %d (%v)", channel, code, got, err)
		}
	}
	if code, err := encodePaint(0); err != nil || code != "" {
		t.Fatalf("encodePaint(0) = %q, %v; want empty", code, err)
	}
}

func TestPaintCodecRefusesUnsupportedForms(t *testing.T) {
	for _, code := range []string{
		"1", "2", "3", "5", "7", // split trees from brush painting
		"10", "C0", "0C", // two characters that are not channel 4+ forms
		"2C2", "8C4", // three or more characters
		"G", "ZZ", // not hexadecimal
	} {
		if _, err := decodePaint(code); ErrorCode(err) != "UNSUPPORTED_PAINT" {
			t.Fatalf("decodePaint(%q) code = %s (%v), want UNSUPPORTED_PAINT", code, ErrorCode(err), err)
		}
	}
}

func TestEncodePaintRefusesChannelBeyondLimit(t *testing.T) {
	if _, err := encodePaint(maxPaintChannel + 1); ErrorCode(err) != "UNSUPPORTED_PAINT_CHANNEL" {
		t.Fatalf("code = %s (%v), want UNSUPPORTED_PAINT_CHANNEL", ErrorCode(err), err)
	}
}
