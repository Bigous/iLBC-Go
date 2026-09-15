package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"io"
	"testing"
)

func TestPacketValidation(t *testing.T) {
	var raw [maxDatagram]byte
	for kind := byte(kindHello); kind <= kindBye; kind++ {
		want := packet{kind: kind, session: 12345, seq: 0xfffffffe}
		if kind == kindAudio {
			for i := range want.audio {
				want.audio[i] = byte(i)
			}
		}
		n := want.marshal(raw[:])
		if got, ok := parsePacket(raw[:n]); !ok || got != want {
			t.Fatalf("round trip: got %+v, valid=%v", got, ok)
		}
		for length := 0; length < n; length++ {
			if _, ok := parsePacket(raw[:length]); ok {
				t.Fatalf("accepted truncated packet of length %d", length)
			}
		}
		if _, ok := parsePacket(append(append([]byte(nil), raw[:n]...), 0)); ok {
			t.Fatal("accepted oversized packet")
		}
	}
	for _, index := range []int{0, 4, 5, 6, 7} {
		p := packet{kind: kindAudio, session: 1}
		p.marshal(raw[:])
		raw[index] = 255
		if _, ok := parsePacket(raw[:]); ok {
			t.Fatalf("accepted invalid header at %d", index)
		}
	}
	(packet{kind: kindHello}).marshal(raw[:])
	if _, ok := parsePacket(raw[:headerBytes]); ok {
		t.Fatal("accepted zero session")
	}
}

func TestJitterReorderingLossAndWrap(t *testing.T) {
	var j jitterBuffer
	if _, _, active := j.take(); active {
		t.Fatal("started without audio")
	}
	first := uint32(0xfffffffe)
	for _, offset := range []uint32{0, 2, 1, 2, 4} {
		j.insert(packet{seq: first + offset, audio: [frameBytes]byte{byte(offset + 1)}})
	}
	for i := 0; i < 3; i++ {
		if _, _, active := j.take(); active {
			t.Fatal("did not buffer initial jitter")
		}
	}
	for i := uint32(0); i < 5; i++ {
		p, received, active := j.take()
		if !active || received != (i != 3) {
			t.Fatalf("frame %d: received=%v active=%v", i, received, active)
		}
		if received && (p.seq != first+i || p.audio[0] != byte(i+1)) {
			t.Fatalf("frame %d reordered incorrectly", i)
		}
		j.insert(packet{seq: first + i}) // Late duplicate must not be replayed.
	}
	if _, received, active := j.take(); received || !active {
		t.Fatal("missing frame should use concealment")
	}
	j.insert(packet{seq: 500})
	if j.next != 500 || j.wait != 3 {
		t.Fatal("large gap did not reset bounded buffer")
	}
}

func TestAudioBridgePartialFramesAndBackpressure(t *testing.T) {
	b := newAudioBridge()
	var pcm [frameSamples * 2]byte
	for i := 0; i < frameSamples; i++ {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(i*100))
	}
	b.process(nil, pcm[:100], 0)
	if len(b.captured) != 0 {
		t.Fatal("sent partial frame")
	}
	b.process(nil, pcm[100:], 0)
	f := <-b.captured
	if f.seq != 0 || f.samples[159] != 15900 {
		t.Fatal("capture frame assembled incorrectly")
	}
	b.playback <- f
	var out [frameSamples*2 + 80]byte
	b.process(out[:100], nil, 0)
	b.process(out[100:], nil, 0)
	if !bytes.Equal(out[:len(pcm)], pcm[:]) || !bytes.Equal(out[len(pcm):], make([]byte, 80)) {
		t.Fatal("playback samples or underrun silence incorrect")
	}
	for i := 0; i < 8; i++ {
		b.process(nil, pcm[:], 0)
	}
	if len(b.captured) != cap(b.captured) || b.capture.seq != 9 {
		t.Fatal("capture queue must be bounded and dropped frames must advance sequence")
	}
	b.fail(io.EOF)
	b.fail(io.ErrClosedPipe)
	if <-b.errors != io.EOF {
		t.Fatal("first audio error not preserved")
	}
}

func TestOptions(t *testing.T) {
	for _, args := range [][]string{{"--serve", "9000"}, {"--connect", "127.0.0.1:9000"}, {"--connect", "[::1]:9000"}} {
		if _, err := parseOptions(args, io.Discard); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{nil, {"--serve", "0"}, {"--serve", "65536"}, {"--serve", "x"}, {"--connect", "localhost:9000"}, {"--connect", "127.0.0.1"}, {"--serve", "9", "--connect", "127.0.0.1:9"}, {"--serve", "9", "extra"}, {"--unknown"}} {
		if _, err := parseOptions(args, io.Discard); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	var help bytes.Buffer
	if _, err := parseOptions([]string{"--help"}, &help); err != flag.ErrHelp || help.Len() == 0 {
		t.Fatal("missing help")
	}
}

func FuzzPacket(f *testing.F) {
	var data [maxDatagram]byte
	for kind := byte(kindHello); kind <= kindBye; kind++ {
		n := (packet{kind: kind, session: 1}).marshal(data[:])
		f.Add(append([]byte(nil), data[:n]...))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		p, ok := parsePacket(data)
		if ok {
			var encoded [maxDatagram]byte
			n := p.marshal(encoded[:])
			got, valid := parsePacket(encoded[:n])
			if !valid || got != p {
				t.Fatal("valid packet failed round trip")
			}
		}
	})
}
