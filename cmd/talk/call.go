package main

import (
	"context"
	"sync"
	"time"

	ilbc "github.com/Bigous/iLBC-Go/src"
)

// Audio devices are injected so transport/codec integration tests do not open
// microphones or speakers on developer or CI machines.
func runCall(ctx context.Context, p *peer, openAudio func(*audioBridge) (func(), error)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	bridge := newAudioBridge()
	stopAudio, err := openAudio(bridge)
	if err != nil {
		_ = p.send(packet{kind: kindBye})
		return err
	}
	defer stopAudio()
	incoming := make(chan packet, 64)
	failures := make(chan error, 3)
	var workers sync.WaitGroup
	start := func(work func() error) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			failures <- work()
		}()
	}
	start(func() error { return p.receive(ctx, incoming) })
	start(func() error {
		encoder, _ := ilbc.NewEncoder(ilbc.Mode20)
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case frame := <-bridge.captured:
				message := packet{kind: kindAudio, seq: frame.seq}
				if err := encoder.Encode(message.audio[:], frame.samples[:]); err != nil {
					return err
				}
				if err := p.send(message); err != nil {
					return err
				}
			}
		}
	})
	start(func() error {
		decoder, _ := ilbc.NewDecoder(ilbc.Mode20, true)
		var jitter jitterBuffer
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case message := <-incoming:
				jitter.insert(message)
			case <-ticker.C:
				message, received, active := jitter.take()
				var frame pcmFrame
				if active {
					if received {
						_ = decoder.Decode(frame.samples[:], message.audio[:])
					} else {
						_ = decoder.Conceal(frame.samples[:])
					}
				}
				select {
				case bridge.playback <- frame:
				default:
				}
			}
		}
	})
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-failures:
	case err = <-bridge.errors:
	}
	cancel()
	_ = p.send(packet{kind: kindBye})
	_ = p.conn.Close()
	workers.Wait()
	return err
}
