package prusa3mf

import "fmt"

// maxPaintChannel is the highest filament channel a single-level paint code names.
const maxPaintChannel = 18

// Paint codes are the per-triangle segmentation values written as paint_color
// by Bambu/Orca and as slic3rpe:mmu_segmentation by PrusaSlicer. Only these
// single-level forms are accepted:
//
//	""  or "0"          not painted
//	"4", "8", "C"       channels 1, 2, 3 (one nibble: channel << 2)
//	"1C" through "FC"   channels 4 through 18 (high nibble = channel-3, then the
//	                    marker nibble C)
//
// Any other code, including multi-level split trees from brush painting, is
// refused with UNSUPPORTED_PAINT. The one-nibble forms and the fixture's "2C"
// (channel 5) are the basis of this reading. The two-character form for
// channels 4 and above has not been checked against a PrusaSlicer-written file.
func decodePaint(code string) (int, error) {
	switch len(code) {
	case 0:
		return 0, nil
	case 1:
		if v, ok := hexNibble(code[0]); ok && v&0x3 == 0 {
			return int(v >> 2), nil
		}
	case 2:
		high, okHigh := hexNibble(code[0])
		low, okLow := hexNibble(code[1])
		if okHigh && okLow && low == 0xC && high != 0 {
			return 3 + int(high), nil
		}
	}
	return 0, fail("UNSUPPORTED_PAINT", "paint_color %q is not a supported single-level segmentation code", code)
}

// encodePaint is the inverse of decodePaint for channels 0 through maxPaintChannel.
func encodePaint(channel int) (string, error) {
	switch {
	case channel == 0:
		return "", nil
	case channel >= 1 && channel <= 3:
		return fmt.Sprintf("%X", channel<<2), nil
	case channel > 3 && channel <= maxPaintChannel:
		return fmt.Sprintf("%XC", channel-3), nil
	}
	return "", fail("UNSUPPORTED_PAINT_CHANNEL", "filament channel %d cannot be painted; the limit is %d", channel, maxPaintChannel)
}

func hexNibble(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	}
	return 0, false
}
