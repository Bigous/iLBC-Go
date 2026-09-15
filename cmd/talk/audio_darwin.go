package main

import (
	"fmt"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego"
)

type audioDescription struct {
	rate                                           float64
	format, flags, bytesPerPacket, framesPerPacket uint32
	bytesPerFrame, channels, bits, reserved        uint32
}

type audioQueueBuffer struct {
	capacity       uint32
	data           *byte
	size           uint32
	user           uintptr
	packetCapacity uint32
	packets        uintptr
	packetCount    uint32
}

func startAudio(bridge *audioBridge) (func(), error) {
	lib, err := purego.Dlopen("/System/Library/Frameworks/AudioToolbox.framework/AudioToolbox", purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return nil, fmt.Errorf("load AudioToolbox: %w", err)
	}
	// Keep the system framework loaded: registered callback trampolines live
	// for the process lifetime. No third-party native library is loaded.
	var newInput, newOutput func(*audioDescription, uintptr, uintptr, uintptr, uintptr, uint32, *uintptr) int32
	var allocate func(uintptr, uint32, **audioQueueBuffer) int32
	var enqueue func(uintptr, *audioQueueBuffer, uint32, uintptr) int32
	var start func(uintptr, uintptr) int32
	var dispose func(uintptr, bool) int32
	purego.RegisterLibFunc(&newInput, lib, "AudioQueueNewInput")
	purego.RegisterLibFunc(&newOutput, lib, "AudioQueueNewOutput")
	purego.RegisterLibFunc(&allocate, lib, "AudioQueueAllocateBuffer")
	purego.RegisterLibFunc(&enqueue, lib, "AudioQueueEnqueueBuffer")
	purego.RegisterLibFunc(&start, lib, "AudioQueueStart")
	purego.RegisterLibFunc(&dispose, lib, "AudioQueueDispose")
	format := audioDescription{rate: 8000, format: 0x6c70636d, flags: 12, bytesPerPacket: 2, framesPerPacket: 1, bytesPerFrame: 2, channels: 1, bits: 16}
	var stopping atomic.Bool
	check := func(operation string, status int32) error {
		if status != 0 {
			return fmt.Errorf("%s: OSStatus %d (check default devices and microphone permission for your terminal)", operation, status)
		}
		return nil
	}
	inputCallback := purego.NewCallback(func(_ uintptr, queue uintptr, buffer *audioQueueBuffer, _ uintptr, _ uint32, _ uintptr) {
		if stopping.Load() {
			return
		}
		bridge.process(nil, unsafe.Slice(buffer.data, int(min(buffer.size, buffer.capacity))), 0)
		if err := check("enqueue microphone buffer", enqueue(queue, buffer, 0, 0)); err != nil {
			bridge.fail(err)
		}
	})
	outputCallback := purego.NewCallback(func(_ uintptr, queue uintptr, buffer *audioQueueBuffer) {
		if stopping.Load() {
			return
		}
		bridge.process(unsafe.Slice(buffer.data, frameSamples*2), nil, 0)
		buffer.size = frameSamples * 2
		if err := check("enqueue output buffer", enqueue(queue, buffer, 0, 0)); err != nil {
			bridge.fail(err)
		}
	})
	var input, output uintptr
	closeAudio := func() {
		stopping.Store(true)
		if input != 0 {
			dispose(input, true)
		}
		if output != 0 {
			dispose(output, true)
		}
	}
	if err := check("open default microphone", newInput(&format, inputCallback, 0, 0, 0, 0, &input)); err != nil {
		return nil, err
	}
	if err := check("open default output", newOutput(&format, outputCallback, 0, 0, 0, 0, &output)); err != nil {
		closeAudio()
		return nil, err
	}
	for _, queue := range []uintptr{input, output} {
		for i := 0; i < 3; i++ {
			var buffer *audioQueueBuffer
			if err := check("allocate audio buffer", allocate(queue, frameSamples*2, &buffer)); err != nil {
				closeAudio()
				return nil, err
			}
			clear(unsafe.Slice(buffer.data, frameSamples*2))
			buffer.size = frameSamples * 2
			if err := check("enqueue audio buffer", enqueue(queue, buffer, 0, 0)); err != nil {
				closeAudio()
				return nil, err
			}
		}
		if err := check("start audio", start(queue, 0)); err != nil {
			closeAudio()
			return nil, err
		}
	}
	return closeAudio, nil
}
