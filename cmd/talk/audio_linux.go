package main

import (
	"fmt"
	"time"

	"github.com/jfreymuth/pulse"
	"github.com/jfreymuth/pulse/proto"
)

type pulsePCM struct{ bridge *audioBridge }

func (p pulsePCM) Format() byte { return proto.FormatInt16LE }
func (p pulsePCM) Read(data []byte) (int, error) {
	p.bridge.process(data, nil, 0)
	return len(data), nil
}
func (p pulsePCM) Write(data []byte) (int, error) {
	p.bridge.process(nil, data, 0)
	return len(data), nil
}

func startAudio(bridge *audioBridge) (func(), error) {
	client, err := pulse.NewClient(pulse.ClientTimeout(3 * time.Second))
	if err != nil {
		return nil, fmt.Errorf("connect to PulseAudio/PipeWire audio service: %w", err)
	}
	stream := pulsePCM{bridge}
	capture, err := client.NewRecord(stream, pulse.RecordMono, pulse.RecordSampleRate(8000), pulse.RecordLatency(0.02))
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("open default microphone: %w", err)
	}
	playback, err := client.NewPlayback(stream, pulse.PlaybackMono, pulse.PlaybackSampleRate(8000), pulse.PlaybackLatency(0.04))
	if err != nil {
		capture.Close()
		client.Close()
		return nil, fmt.Errorf("open default audio output: %w", err)
	}
	playback.Start()
	capture.Start()
	return func() {
		capture.Close()
		playback.Close()
		client.Close()
	}, nil
}
