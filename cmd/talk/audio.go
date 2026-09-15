package main

import "encoding/binary"

type pcmFrame struct {
	seq     uint32
	samples [frameSamples]int16
}

// Only the audio callback accesses capture/current offsets. Channel transfers
// copy complete frames, so neither native audio memory nor PCM is shared.
type audioBridge struct {
	captured chan pcmFrame
	playback chan pcmFrame
	errors   chan error
	capture  pcmFrame
	output   pcmFrame
	inputAt  int
	outputAt int
}

func newAudioBridge() *audioBridge {
	return &audioBridge{
		captured: make(chan pcmFrame, 4),
		playback: make(chan pcmFrame, 3),
		errors:   make(chan error, 1),
		outputAt: frameSamples,
	}
}

func (b *audioBridge) fail(err error) {
	select {
	case b.errors <- err:
	default:
	}
}

// process never waits for network, codec work, or an empty/full channel.
// The backend supplies mono signed 16-bit little-endian PCM at 8 kHz.
func (b *audioBridge) process(output, input []byte, _ uint32) {
	for i := 0; i+1 < len(input); i += 2 {
		b.capture.samples[b.inputAt] = int16(binary.LittleEndian.Uint16(input[i:]))
		b.inputAt++
		if b.inputAt == frameSamples {
			select {
			case b.captured <- b.capture:
			default: // Drop a frame rather than stall the audio device.
			}
			b.capture.seq++
			b.inputAt = 0
		}
	}
	clear(output)
	for i := 0; i+1 < len(output); i += 2 {
		if b.outputAt == frameSamples {
			select {
			case b.output = <-b.playback:
				b.outputAt = 0
			default:
				return // Leave the remaining output silent on underrun.
			}
		}
		binary.LittleEndian.PutUint16(output[i:], uint16(b.output.samples[b.outputAt]))
		b.outputAt++
	}
}
