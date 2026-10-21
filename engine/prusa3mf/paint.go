package prusa3mf

// maxPaintState is the highest filament state a paint leaf can name. States 0
// to 2 use two bits; state 3 is an escape followed by a four-bit extension, so
// the largest encodable state is 3 + 15.
const maxPaintState = 18

// maxPaintCodeLen bounds one paint_color value. The tree walk recurses once per
// split node, so the bound also bounds recursion depth.
const maxPaintCodeLen = 4096

// paintCode is one distinct paint_color value and the filament channels it names.
type paintCode struct {
	code     string
	channels uint32 // bit N set when a leaf paints channel N
}

// decodePaint validates a paint_color value and returns the channels its leaves
// name. The value is a TriangleSelector bitstream, the format Bambu Studio forked
// from PrusaSlicer, read as in Panda2Prusa's paint.py: hex digits form a bit
// stream read from the last character backwards, least significant bit first.
//
// A node is a leaf (split bits 00, then state:2, plus a four-bit extension when
// state is 3) or a split (split bits 1 to 3, a side index, then split+1 child
// nodes). Bits after the root node must be zero. Leaves with state 0 mean "use
// the volume's extruder", so they name no channel and are not in the mask.
//
// Nothing is remapped: the writer emits the original code unchanged.
func decodePaint(code string) (uint32, error) {
	if code == "" {
		return 0, nil
	}
	if len(code) > maxPaintCodeLen {
		return 0, fail("UNSUPPORTED_PAINT", "paint_color is longer than %d characters", maxPaintCodeLen)
	}
	bits := make([]uint8, 0, 4*len(code))
	for i := len(code) - 1; i >= 0; i-- {
		v, ok := hexNibble(code[i])
		if !ok {
			return 0, fail("UNSUPPORTED_PAINT", "paint_color %q contains a non-hexadecimal character", code)
		}
		for b := 0; b < 4; b++ {
			bits = append(bits, (v>>b)&1)
		}
	}
	r := &paintReader{bits: bits}
	if !r.node() {
		return 0, fail("UNSUPPORTED_PAINT", "paint_color %q is truncated", code)
	}
	for _, b := range r.bits[r.pos:] {
		if b != 0 {
			return 0, fail("UNSUPPORTED_PAINT", "paint_color %q has data after its root node", code)
		}
	}
	return r.channels, nil
}

// paintReader walks a paint bitstream. Every method reports false when the
// stream ends early, so truncated codes are refused rather than read as zeros.
type paintReader struct {
	bits     []uint8
	pos      int
	channels uint32
}

// take reads n bits, least significant first.
func (r *paintReader) take(n int) (int, bool) {
	if r.pos+n > len(r.bits) {
		return 0, false
	}
	v := 0
	for i := 0; i < n; i++ {
		v |= int(r.bits[r.pos+i]) << i
	}
	r.pos += n
	return v, true
}

// node reads one leaf, or one split node together with all of its children.
func (r *paintReader) node() bool {
	split, ok := r.take(2)
	if !ok {
		return false
	}
	if split == 0 {
		state, ok := r.take(2)
		if !ok {
			return false
		}
		if state == 3 {
			ext, ok := r.take(4)
			if !ok {
				return false
			}
			state = ext + 3
		}
		if state != 0 {
			r.channels |= 1 << state
		}
		return true
	}
	if _, ok := r.take(2); !ok { // special side index
		return false
	}
	for i := 0; i <= split; i++ {
		if !r.node() {
			return false
		}
	}
	return true
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
