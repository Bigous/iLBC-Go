package ilbc

import "math"

// flushSubnormalHistory clears nonzero float32 magnitudes below 2^-126.
// Integer bit tests avoid performing floating-point operations on subnormal
// operands. Normal values, signed zeros, infinities and NaNs are unchanged.
// This is local to decoder history; it does not change CPU floating-point modes.
func flushSubnormalHistory(history []float32) {
	for i, value := range history {
		magnitude := math.Float32bits(value) & 0x7fffffff
		if magnitude != 0 && magnitude < 0x00800000 {
			history[i] = 0
		}
	}
}
