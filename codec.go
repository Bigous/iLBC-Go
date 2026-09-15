// Package ilbc implements the floating-point iLBC speech codec from RFC 3951.
// Audio is mono, signed 16-bit PCM at 8000 Hz. Each call processes one frame;
// transport headers, resampling, and audio file containers are outside its scope.
// Instances retain stream history and must not be used concurrently. Independent
// instances can be used concurrently. The implementation uses neither CGo nor unsafe.
package ilbc

import "errors"

// SampleRate is the required PCM sample rate in Hz.
const SampleRate = 8000

// Mode selects the duration of each encoded frame.
type Mode int

const (
	// Mode20 encodes 160 samples into 38 bytes (15.2 kbit/s).
	Mode20 Mode = 20
	// Mode30 encodes 240 samples into 50 bytes (13.33 kbit/s).
	Mode30 Mode = 30
)

var (
	ErrInvalidMode   = errors.New("ilbc: mode must be Mode20 or Mode30")
	ErrUninitialized = errors.New("ilbc: instance must be created with its constructor")
	ErrPCMSize       = errors.New("ilbc: PCM input must contain exactly one frame")
	ErrPacketSize    = errors.New("ilbc: packet must contain exactly one frame")
	ErrShortBuffer   = errors.New("ilbc: output buffer is too short")
)

// Encoder holds the history of one PCM stream. Use NewEncoder to initialize it.
// Do not copy an Encoder after first use.
type Encoder struct{ state encoderState }

// NewEncoder creates an encoder for the selected frame duration.
func NewEncoder(mode Mode) (*Encoder, error) {
	if mode != Mode20 && mode != Mode30 {
		return nil, ErrInvalidMode
	}
	e := new(Encoder)
	initEncode(&e.state, int(mode))
	return e, nil
}

// FrameSamples returns the number of PCM samples consumed per frame.
func (e *Encoder) FrameSamples() int { return e.state.frameSamples }

// FrameBytes returns the number of bytes produced per frame.
func (e *Encoder) FrameBytes() int { return e.state.frameBytes }

// Encode writes one iLBC frame to dst. pcm must have exactly FrameSamples()
// samples, and dst must have at least FrameBytes() bytes. The rest of dst is
// untouched. Validation errors leave both stream history and dst unchanged.
// After construction this method performs no heap allocations.
func (e *Encoder) Encode(dst []byte, pcm []int16) error {
	if e.state.mode == 0 {
		return ErrUninitialized
	}
	if len(pcm) != e.state.frameSamples {
		return ErrPCMSize
	}
	if len(dst) < e.state.frameBytes {
		return ErrShortBuffer
	}
	var block [240]float32
	for i, sample := range pcm {
		block[i] = float32(sample)
	}
	encodeFrame(span[byte]{data: dst[:e.state.frameBytes]}, span[float32]{data: block[:]}, &e.state)
	return nil
}

// Reset discards stream history while retaining the selected mode.
func (e *Encoder) Reset() error {
	if e.state.mode == 0 {
		return ErrUninitialized
	}
	initEncode(&e.state, e.state.mode)
	return nil
}

// Decoder holds the history of one iLBC stream. Use NewDecoder to initialize it.
// Do not copy a Decoder after first use.
type Decoder struct{ state decoderState }

// NewDecoder creates a decoder. enhance enables the RFC's pitch-synchronous
// enhancement, improving quality at additional CPU cost and algorithmic delay.
func NewDecoder(mode Mode, enhance bool) (*Decoder, error) {
	if mode != Mode20 && mode != Mode30 {
		return nil, ErrInvalidMode
	}
	d := new(Decoder)
	var enhancement int
	if enhance {
		enhancement = 1
	}
	initDecode(&d.state, int(mode), enhancement)
	return d, nil
}

// FrameSamples returns the number of PCM samples produced per frame.
func (d *Decoder) FrameSamples() int { return d.state.frameSamples }

// FrameBytes returns the required size of an encoded frame.
func (d *Decoder) FrameBytes() int { return d.state.frameBytes }

// Decode writes one frame of PCM to dst. packet must have exactly FrameBytes()
// bytes and dst must have at least FrameSamples() samples. The rest of dst is
// untouched. Frames marked empty or with an invalid start index are concealed
// as specified in the RFC. Validation errors leave history and dst unchanged.
// After construction this method performs no heap allocations.
func (d *Decoder) Decode(dst []int16, packet []byte) error {
	if d.state.mode == 0 {
		return ErrUninitialized
	}
	if len(packet) != d.state.frameBytes {
		return ErrPacketSize
	}
	if len(dst) < d.state.frameSamples {
		return ErrShortBuffer
	}
	d.decode(dst, packet, 1)
	return nil
}

// Conceal generates one frame for a missing packet and updates stream history.
// Call it once for each missing frame. dst must hold at least FrameSamples()
// samples. After construction this method performs no heap allocations.
func (d *Decoder) Conceal(dst []int16) error {
	if d.state.mode == 0 {
		return ErrUninitialized
	}
	if len(dst) < d.state.frameSamples {
		return ErrShortBuffer
	}
	d.decode(dst, nil, 0)
	return nil
}

func (d *Decoder) decode(dst []int16, packet []byte, received int) {
	// Recursive filters can retain subnormal tails after the PCM output has
	// become silent. Clear these between frames to avoid costly arithmetic.
	flushSubnormalHistory(d.state.syntMem[:])
	flushSubnormalHistory(d.state.hpomem[:])
	var block [240]float32
	decodeFrame(span[float32]{data: block[:]}, span[byte]{data: packet}, &d.state, received)
	var sample float32
	for i := 0; i < d.state.frameSamples; i++ {
		sample = block[i]
		if sample < -32768 {
			sample = -32768
		} else if sample > 32767 {
			sample = 32767
		}
		dst[i] = int16(sample)
	}
}

// Reset discards stream history, retaining the mode and enhancement setting.
func (d *Decoder) Reset() error {
	if d.state.mode == 0 {
		return ErrUninitialized
	}
	initDecode(&d.state, d.state.mode, d.state.enhancement)
	return nil
}
