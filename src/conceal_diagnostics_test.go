package ilbc

import (
	"fmt"
	"math"
	"testing"
)

// Snapshot helpers isolate histories for diagnostics. Production also clears subnormal filter tails.
func diagnosticSubnormal(v float32) bool {
	bits := math.Float32bits(v) & 0x7fffffff
	return bits != 0 && bits < 0x00800000
}

func diagnosticHistories(s *decoderState) [][]float32 {
	return [][]float32{s.syntMem[:], s.hpomem[:], s.prevResidual[:], s.enhBuf[:]}
}

func diagnosticSubnormalCounts(s *decoderState) [4]int {
	var counts [4]int
	for group, values := range diagnosticHistories(s) {
		for _, value := range values {
			if diagnosticSubnormal(value) {
				counts[group]++
			}
		}
	}
	return counts
}

func diagnosticFlushHistories(s *decoderState) {
	for _, values := range diagnosticHistories(s) {
		for i, value := range values {
			if diagnosticSubnormal(value) {
				values[i] = 0
			}
		}
	}
}

func diagnosticDecoder(tb testing.TB, mode Mode, enhance bool, losses int) decoderState {
	tb.Helper()
	e, err := NewEncoder(mode)
	if err != nil {
		tb.Fatal(err)
	}
	d, err := NewDecoder(mode, enhance)
	if err != nil {
		tb.Fatal(err)
	}
	pcm := make([]int16, e.FrameSamples())
	out := make([]int16, len(pcm))
	packet := make([]byte, e.FrameBytes())
	for i := range pcm {
		pcm[i] = int16((i*793)%32000 - 16000)
	}
	for frame := 0; frame < 12; frame++ {
		if err := e.Encode(packet, pcm); err != nil {
			tb.Fatal(err)
		}
		if err := d.Decode(out, packet); err != nil {
			tb.Fatal(err)
		}
	}
	for frame := 0; frame < losses; frame++ {
		if err := d.Conceal(out); err != nil {
			tb.Fatal(err)
		}
	}
	// Private state is arrays/scalars plus an immutable layout pointer, so these
	// internal benchmark snapshots do not share mutable stream history.
	return d.state
}

func TestConcealSubnormalDiagnostics(t *testing.T) {
	for _, mode := range []Mode{Mode20, Mode30} {
		for _, enhance := range []bool{false, true} {
			for _, losses := range []int{0, 8, 32, 512} {
				s := diagnosticDecoder(t, mode, enhance, losses)
				counts := diagnosticSubnormalCounts(&s)
				t.Logf("mode=%d enhance=%v losses=%d subnormals: synthesis=%d highpass=%d residual=%d enhancer=%d",
					mode, enhance, losses, counts[0], counts[1], counts[2], counts[3])
				// Classification varies by platform/state; the diagnostic makes no
				// performance assertion. Verify its experimental clearing operation.
				diagnosticFlushHistories(&s)
				if diagnosticSubnormalCounts(&s) != [4]int{} {
					t.Fatal("diagnostic failed to clear subnormal histories")
				}
			}
		}
	}
}

func BenchmarkConcealScenarios(b *testing.B) {
	for _, mode := range []Mode{Mode20, Mode30} {
		for _, enhance := range []bool{false, true} {
			for _, losses := range []int{0, 8, 512} {
				original := diagnosticDecoder(b, mode, enhance, losses)
				for _, flush := range []bool{false, true} {
					label := fmt.Sprintf("mode%d/enhance%v/losses%d/flush%v", mode, enhance, losses, flush)
					b.Run(label, func(b *testing.B) {
						baseline := original
						if flush {
							diagnosticFlushHistories(&baseline)
						}
						var d Decoder
						var out [240]int16
						b.ReportAllocs()
						b.ResetTimer()
						for i := 0; i < b.N; i++ {
							// Both variants include the same snapshot copy cost. Each
							// iteration processes exactly the selected loss position.
							// This differs from BenchmarkCodec's evolving stream.
							d.state = baseline
							if err := d.Conceal(out[:baseline.frameSamples]); err != nil {
								b.Fatal(err)
							}
						}
					})
				}
			}
		}
	}
}
