package ilbc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"testing"
)

func readData(t *testing.T, file string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + file)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestReference(t *testing.T) {
	for _, mode := range []Mode{Mode20, Mode30} {
		for _, enhance := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/enhance=%v", mode, enhance), func(t *testing.T) {
				e, _ := NewEncoder(mode)
				d, _ := NewDecoder(mode, enhance)
				n, size := e.FrameSamples(), e.FrameBytes()
				input := readData(t, fmt.Sprintf("input%d.pcm", mode))
				bits := readData(t, fmt.Sprintf("encoded%d.bin", mode))
				want := readData(t, fmt.Sprintf("decoded%d_%v.pcm", mode, enhance))
				loss := readData(t, fmt.Sprintf("loss%d.bin", mode))
				pcm := make([]int16, n)
				out := make([]int16, n)
				packet := make([]byte, size)
				var maxDelta, different int
				for frame, received := range loss {
					for j := range pcm {
						pcm[j] = int16(binary.LittleEndian.Uint16(input[(frame*n+j)*2:]))
					}
					if err := e.Encode(packet, pcm); err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(packet, bits[frame*size:(frame+1)*size]) {
						t.Fatalf("encoded mismatch at frame %d: got %x want %x", frame, packet, bits[frame*size:(frame+1)*size])
					}
					var err error
					if received == 0 {
						err = d.Conceal(out)
					} else {
						err = d.Decode(out, packet)
					}
					if err != nil {
						t.Fatal(err)
					}
					for j, v := range out {
						w := int16(binary.LittleEndian.Uint16(want[(frame*n+j)*2:]))
						delta := int(v) - int(w)
						if delta < 0 {
							delta = -delta
						}
						if delta > 0 {
							different++
						}
						if delta > maxDelta {
							maxDelta = delta
						}
						if delta > 1 {
							t.Fatalf("decoded mismatch frame=%d sample=%d: got %d want %d", frame, j, v, w)
						}
					}
				}
				t.Logf("PCM max delta=%d; differing samples=%d", maxDelta, different)
			})
		}
	}
}

func BenchmarkCodec(b *testing.B) {
	for _, mode := range []Mode{Mode20, Mode30} {
		e, _ := NewEncoder(mode)
		d, _ := NewDecoder(mode, true)
		pcm := make([]int16, e.FrameSamples())
		packet := make([]byte, e.FrameBytes())
		for i := range pcm {
			pcm[i] = int16((i*793)%32000 - 16000)
		}
		_ = e.Encode(packet, pcm)
		b.Run(fmt.Sprintf("Encode%d", mode), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = e.Encode(packet, pcm)
			}
		})
		b.Run(fmt.Sprintf("Decode%d", mode), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = d.Decode(pcm, packet)
			}
		})
		b.Run(fmt.Sprintf("Conceal%d", mode), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = d.Conceal(pcm)
			}
		})
	}
}
