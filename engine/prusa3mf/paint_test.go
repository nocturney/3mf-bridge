package prusa3mf

import (
	"strings"
	"testing"
)

// channelMask is the decodePaint result for the given channels.
func channelMask(channels ...int) uint32 {
	var mask uint32
	for _, ch := range channels {
		mask |= 1 << ch
	}
	return mask
}

// TestDecodePaintSingleLevelCodes pins the one-level codes and the fixture's "2C".
func TestDecodePaintSingleLevelCodes(t *testing.T) {
	cases := map[string]uint32{
		"":   0,
		"0":  0,
		"4":  channelMask(1),
		"8":  channelMask(2),
		"0C": channelMask(3),
		"1C": channelMask(4),
		"2C": channelMask(5),
		"2c": channelMask(5),
		"FC": channelMask(18),
	}
	for code, want := range cases {
		got, err := decodePaint(code)
		if err != nil {
			t.Fatalf("decodePaint(%q) error: %v", code, err)
		}
		if got != want {
			t.Fatalf("decodePaint(%q) = %b, want %b", code, got, want)
		}
	}
}

// TestDecodePaintNestedTrees uses codes generated from the bitstream grammar in
// Panda2Prusa's paint.py and checked with its decoder. Split trees must decode to
// the same channels, and the writer keeps the codes byte for byte.
func TestDecodePaintNestedTrees(t *testing.T) {
	cases := map[string]uint32{
		// Two-way split into two leaves.
		"841": channelMask(1, 2),
		// Three-way split; the state 0 leaf names no channel.
		"0CFC2C906": channelMask(3, 5, 18),
		// Four-way split with a nested three-way split.
		"84C04201CF": channelMask(1, 2, 4, 7),
	}
	for code, want := range cases {
		got, err := decodePaint(code)
		if err != nil {
			t.Fatalf("decodePaint(%q) error: %v", code, err)
		}
		if got != want {
			t.Fatalf("decodePaint(%q) = %b, want %b", code, got, want)
		}
	}
}

func TestDecodePaintRefusesMalformedCodes(t *testing.T) {
	// Split nodes whose children are missing, a state 3 leaf without its
	// extension, and a split node whose children run out.
	truncated := []string{"1", "2", "3", "5", "7", "81", "C", "2C2"}
	// Non-zero bits after the root node.
	trailing := []string{"10", "C0", "40", "1C0", "8410", "8C4", "84C04201C"}
	// Not hexadecimal, and longer than the allowed bound.
	invalid := []string{"G", "ZZ", strings.Repeat("0", maxPaintCodeLen+1)}
	for _, code := range append(append(truncated, trailing...), invalid...) {
		if _, err := decodePaint(code); ErrorCode(err) != "UNSUPPORTED_PAINT" {
			t.Fatalf("decodePaint(%q) code = %s (%v), want UNSUPPORTED_PAINT", code, ErrorCode(err), err)
		}
	}
}

func TestDecodePaintAcceptsLongestAllowedCode(t *testing.T) {
	got, err := decodePaint(strings.Repeat("0", maxPaintCodeLen))
	if err != nil || got != 0 {
		t.Fatalf("decodePaint(longest allowed) = %b, %v; want 0, nil", got, err)
	}
}
