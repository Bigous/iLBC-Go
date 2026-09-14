package ilbc

import (
	"math"
	"reflect"
	"testing"
)

func TestBitPacking(t *testing.T) {
	for width := 0; width <= 16; width++ {
		for value := 0; value < (1 << width); value++ {
			var original, first, rest [1]int
			original[0] = value
			split := width / 2
			packsplit(span[int]{data: original[:]}, span[int]{data: first[:]}, span[int]{data: rest[:]}, split, width)
			packcombine(span[int]{data: first[:]}, rest[0], width-split)
			if first[0] != value {
				t.Fatalf("split %d/%d", value, width)
			}
			var data [4]byte
			var cursor [1]span[byte]
			var pos, result [1]int
			cursor[0] = span[byte]{data: data[:]}
			pos[0] = 3
			dopack(span[span[byte]]{data: cursor[:]}, value, width, span[int]{data: pos[:]})
			cursor[0] = span[byte]{data: data[:]}
			pos[0] = 3
			unpack(span[span[byte]]{data: cursor[:]}, span[int]{data: result[:]}, width, span[int]{data: pos[:]})
			if result[0] != value {
				t.Fatalf("packed %d/%d -> %d", value, width, result[0])
			}
		}
	}
}

func TestZeroCodebookSearch(t *testing.T) {
	e, _ := NewEncoder(Mode20)
	var index, gains [3]int
	var target [40]float32
	var memory [147]float32
	var weights [11]float32
	var history [10]float32
	weights[0] = 1
	searchCodebook(&e.state, span[int]{data: index[:]}, span[int]{data: gains[:]}, span[float32]{data: target[:]}, span[float32]{data: memory[:]}, 147, 40, 3, span[float32]{data: weights[:]}, span[float32]{data: history[:]}, 1)
	var decoded [40]float32
	constructCodebook(span[float32]{data: decoded[:]}, span[int]{data: index[:]}, span[int]{data: gains[:]}, span[float32]{data: memory[:]}, 147, 40, 3)
	if decoded != target {
		t.Fatal("zero codebook generated nonzero excitation")
	}
}

func TestUpsampleConvolution(t *testing.T) {
	for _, n := range []int{3, 5, 9, 13} {
		x := make([]float32, n)
		y := make([]float32, n*4)
		for i := range x {
			x[i] = float32(i*i - 3*i + 1)
		}
		upsampleEnhancement(span[float32]{data: y}, span[float32]{data: x}, n, 3)
		half := min(3, n/2)
		for i := 0; i < n; i++ {
			for phase := 0; phase < 4; phase++ {
				var want float32
				for k := 0; k < half*2+1; k++ {
					j := i + half - k
					if j >= 0 && j < n {
						want += x[j] * polyphaserTbl[phase*7+3-half+k]
					}
				}
				if y[i*4+phase] != want {
					t.Fatalf("n=%d i=%d phase=%d: %g != %g", n, i, phase, y[i*4+phase], want)
				}
			}
		}
	}
}

func TestEnhancementBoundaries(t *testing.T) {
	var data [160]float32
	var out [80]float32
	var position [1]float32
	for _, estimate := range []float32{0, 80} {
		refiner(span[float32]{data: out[:]}, span[float32]{data: position[:]}, span[float32]{data: data[:]}, 160, 40, estimate, 40)
		if out != [80]float32{} || position[0] < 0 || position[0] > 81 {
			t.Fatal("invalid boundary refinement", position)
		}
	}
	// An opposite-phase surround is collinear with the center. Its Gram
	// determinant is zero, so the constrained smoother must keep the center.
	var sequence [560]float32
	for i := range sequence {
		sequence[i] = -1
	}
	for i := 240; i < 320; i++ {
		sequence[i] = 1
	}
	smath(span[float32]{data: out[:]}, span[float32]{data: sequence[:]}, 3, .05)
	for _, v := range out {
		if v != 1 {
			t.Fatal("degenerate smoother changed center", v)
		}
	}
}

func TestSpectralBoundaries(t *testing.T) {
	values := []float32{-1, 0.2, 0.1, 0.4, 0.8, 1, 1.2, 1.5, 4, 4.1}
	if checkLSF(span[float32]{data: values}, 10, 1) != 1 {
		t.Fatal("expected LSF correction")
	}
	for _, v := range values[:9] {
		if v < .01 || v > 3.14 {
			t.Fatal("unbounded LSF", values)
		}
	}
	for _, endpoints := range [][2]float32{{-1, 4}, {0, 2}, {1, 4}} {
		var lsf [10]float32
		var coeff [11]float32
		for i := range lsf {
			lsf[i] = endpoints[0] + float32(i)*(endpoints[1]-endpoints[0])/9
		}
		lsf2a(span[float32]{data: coeff[:]}, span[float32]{data: lsf[:]})
		if coeff[0] != 1 {
			t.Fatal("LPC leading coefficient")
		}
		for _, v := range coeff {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				t.Fatal("nonfinite LPC")
			}
		}
	}
	if gaindequant(0, 1, 0) != 0 {
		t.Fatal("unknown codebook must return zero")
	}
}

func TestInternalModeGuards(t *testing.T) {
	for _, fn := range []func(){func() { initEncode(new(encoderState), 0) }, func() { initDecode(new(decoderState), 0, 0) }} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("missing mode guard")
				}
			}()
			fn()
		}()
	}
}

func TestConcealmentCorrections(t *testing.T) {
	for _, mode := range []Mode{Mode20, Mode30} {
		for _, enhance := range []bool{false, true} {
			e, _ := NewEncoder(mode)
			d, _ := NewDecoder(mode, enhance)
			d2, _ := NewDecoder(mode, enhance)
			pcm := make([]int16, e.FrameSamples())
			packet := make([]byte, e.FrameBytes())
			out := make([]int16, len(pcm))
			lost := make([]int16, len(pcm))
			for i := range pcm {
				pcm[i] = int16(i * 397)
			}
			_ = e.Encode(packet, pcm)
			_ = d.Decode(out, packet)
			_ = d2.Decode(lost, packet)
			marked := append([]byte(nil), packet...)
			marked[len(marked)-1] |= 1
			_ = d.Decode(out, marked)
			_ = d2.Conceal(lost)
			if !reflect.DeepEqual(out, lost) {
				t.Fatal("lost bit differs from concealment")
			}
			for i := 0; i < 50; i++ {
				_ = d.Conceal(out)
			}
			for _, v := range out {
				if v != 0 {
					t.Fatal("long loss burst did not decay to silence")
				}
			}
			if d.state.consPLICount != 9 {
				t.Fatal("PLC count not saturated")
			}
			for i := 0; i < 10; i++ {
				_ = d.Decode(out, packet)
				if d.state.lastLag < 20 || d.state.lastLag > 119 {
					t.Fatal("pitch outside history", d.state.lastLag)
				}
				_ = d.Conceal(out)
			}
		}
	}
}
