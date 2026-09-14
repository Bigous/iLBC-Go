package ilbc

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"
)

func TestValidation(t *testing.T) {
	for _, m := range []Mode{-1, 0, 10, 21, 40} {
		if e, err := NewEncoder(m); e != nil || err != ErrInvalidMode {
			t.Fatal(e, err)
		}
		if d, err := NewDecoder(m, true); d != nil || err != ErrInvalidMode {
			t.Fatal(d, err)
		}
	}
	var e Encoder
	var d Decoder
	for _, err := range []error{e.Encode(nil, nil), e.Reset(), d.Decode(nil, nil), d.Conceal(nil), d.Reset()} {
		if err != ErrUninitialized {
			t.Fatal(err)
		}
	}
	for _, mode := range []Mode{Mode20, Mode30} {
		enc, _ := NewEncoder(mode)
		dec, _ := NewDecoder(mode, true)
		n, size := enc.FrameSamples(), enc.FrameBytes()
		if n != int(mode)*8 || dec.FrameSamples() != n || dec.FrameBytes() != size {
			t.Fatal("frame dimensions")
		}
		pcm := make([]int16, n)
		packet := bytes.Repeat([]byte{0xAA}, size+3)
		out := make([]int16, n+3)
		for i := range out {
			out[i] = 1234
		}
		es, ds := enc.state, dec.state
		for _, test := range []struct{ err, want error }{
			{enc.Encode(packet, pcm[:n-1]), ErrPCMSize},
			{enc.Encode(packet, append(pcm, 0)), ErrPCMSize},
			{enc.Encode(packet[:size-1], pcm), ErrShortBuffer},
			{dec.Decode(out, packet), ErrPacketSize},
			{dec.Decode(out, packet[:size-1]), ErrPacketSize},
			{dec.Decode(out[:n-1], packet[:size]), ErrShortBuffer},
			{dec.Conceal(out[:n-1]), ErrShortBuffer},
		} {
			if test.err != test.want {
				t.Fatal(test)
			}
		}
		if enc.state != es || dec.state != ds {
			t.Fatal("validation changed history")
		}
		if !bytes.Equal(packet, bytes.Repeat([]byte{0xAA}, size+3)) {
			t.Fatal("validation changed destination")
		}
		for _, v := range out {
			if v != 1234 {
				t.Fatal("validation changed PCM")
			}
		}
		if err := enc.Encode(packet, pcm); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(packet[size:], []byte{0xAA, 0xAA, 0xAA}) {
			t.Fatal("encoder overwrote suffix")
		}
		before := append([]byte(nil), packet...)
		if err := dec.Decode(out, packet[:size]); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, packet) {
			t.Fatal("decoder modified packet")
		}
		if err := dec.Conceal(out); err != nil {
			t.Fatal(err)
		}
		for _, v := range out[n:] {
			if v != 1234 {
				t.Fatal("decoder overwrote suffix")
			}
		}
	}
}

func TestResetAndAllocations(t *testing.T) {
	for _, mode := range []Mode{Mode20, Mode30} {
		for _, enhance := range []bool{false, true} {
			e, _ := NewEncoder(mode)
			d, _ := NewDecoder(mode, enhance)
			freshE, _ := NewEncoder(mode)
			freshD, _ := NewDecoder(mode, enhance)
			pcm := make([]int16, e.FrameSamples())
			out := make([]int16, len(pcm))
			packet := make([]byte, e.FrameBytes())
			want := make([]byte, len(packet))
			for i := range pcm {
				pcm[i] = int16(i * 397)
			}
			for i := 0; i < 12; i++ {
				_ = e.Encode(packet, pcm)
				_ = d.Decode(out, packet)
				_ = d.Conceal(out)
			}
			if err := e.Reset(); err != nil {
				t.Fatal(err)
			}
			if err := d.Reset(); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(e, freshE) || !reflect.DeepEqual(d, freshD) {
				t.Fatal("reset differs from new instance")
			}
			_ = e.Encode(packet, pcm)
			_ = freshE.Encode(want, pcm)
			if !bytes.Equal(packet, want) {
				t.Fatal("reset output differs")
			}
			for name, fn := range map[string]func(){"encode": func() { _ = e.Encode(packet, pcm) }, "decode": func() { _ = d.Decode(out, packet) }, "conceal": func() { _ = d.Conceal(out) }, "reset": func() { _ = e.Reset(); _ = d.Reset() }} {
				if allocs := testing.AllocsPerRun(30, fn); allocs != 0 {
					t.Fatalf("%s allocated %g times", name, allocs)
				}
			}
		}
	}
}

func TestArbitraryPackets(t *testing.T) {
	rng := rand.New(rand.NewSource(3951))
	for _, mode := range []Mode{Mode20, Mode30} {
		for _, enhance := range []bool{false, true} {
			d, _ := NewDecoder(mode, enhance)
			out := make([]int16, d.FrameSamples())
			packet := make([]byte, d.FrameBytes())
			for i := 0; i < 2000; i++ {
				rng.Read(packet)
				if err := d.Decode(out, packet); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

func FuzzDecode(f *testing.F) {
	f.Add([]byte{0}, true)
	f.Add(bytes.Repeat([]byte{255}, 38), false)
	f.Add(make([]byte, 50), true)
	f.Fuzz(func(t *testing.T, packet []byte, enhance bool) {
		mode := Mode20
		if len(packet) == 50 {
			mode = Mode30
		}
		d, _ := NewDecoder(mode, enhance)
		out := make([]int16, d.FrameSamples())
		err := d.Decode(out, packet)
		if len(packet) != d.FrameBytes() {
			if err != ErrPacketSize {
				t.Fatal(err)
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		_ = d.Conceal(out)
		_ = d.Decode(out, packet)
	})
}
