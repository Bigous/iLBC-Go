package ilbc

import (
	"fmt"
	"math"
	"testing"
)

func TestFlushSubnormalHistory(t *testing.T) {
	bits := []uint32{
		0, 0x80000000, // Preserve signed zeros.
		1, 0x80000001, 0x007fffff, 0x807fffff, // Clear both ends of each subnormal range.
		0x00800000, 0x80800000, // Preserve the smallest normal magnitudes.
		0x3f800000, 0xbf800000, 0x7f7fffff, 0xff7fffff,
		0x7f800000, 0xff800000, 0x7fc01234, 0xffc01234, // Preserve special values.
	}
	values := make([]float32, len(bits))
	for i, b := range bits {
		values[i] = math.Float32frombits(b)
	}
	flushSubnormalHistory(nil)
	flushSubnormalHistory(values)
	for i, want := range bits {
		if i >= 2 && i <= 5 {
			want = 0
		}
		if got := math.Float32bits(values[i]); got != want {
			t.Fatalf("element %d: got %08x, want %08x", i, got, want)
		}
	}
}

func TestDecoderSubnormalHistory(t *testing.T) {
	for _, mode := range []Mode{Mode20, Mode30} {
		for _, enhance := range []bool{false, true} {
			state := diagnosticDecoder(t, mode, enhance, 512)
			if state.syntMem != [10]float32{} || state.hpomem != [4]float32{} {
				t.Fatalf("mode=%d enhance=%v: filters retained a tail after long loss", mode, enhance)
			}
			packets := readData(t, fmt.Sprintf("encoded%d.bin", mode))
			packet := packets[16*state.frameBytes : 17*state.frameBytes]
			for _, received := range []bool{false, true} {
				withTail := Decoder{state: state}
				clean := Decoder{state: state}
				for i := range withTail.state.syntMem {
					withTail.state.syntMem[i] = math.Float32frombits(uint32(i+1) | uint32(i%2)<<31)
				}
				for i := range withTail.state.hpomem {
					withTail.state.hpomem[i] = math.Float32frombits(0x007fffff - uint32(i))
				}
				// Validation errors must not clear even subnormal history.
				before := withTail.state
				if withTail.Conceal(nil) != ErrShortBuffer || withTail.Decode(nil, packet) != ErrShortBuffer {
					t.Fatal("unexpected validation error")
				}
				if withTail.state != before {
					t.Fatal("validation changed subnormal history")
				}
				var got, want [240]int16
				var err1, err2 error
				if received {
					err1 = withTail.Decode(got[:], packet)
					err2 = clean.Decode(want[:], packet)
				} else {
					err1 = withTail.Conceal(got[:])
					err2 = clean.Conceal(want[:])
				}
				if err1 != nil || err2 != nil {
					t.Fatal(err1, err2)
				}
				if got != want || withTail.state != clean.state {
					t.Fatalf("mode=%d enhance=%v received=%v: subnormal tail changed decoded output or history", mode, enhance, received)
				}
			}
		}
	}
}
