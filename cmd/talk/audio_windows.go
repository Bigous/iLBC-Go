package main

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var winmm = syscall.NewLazyDLL("winmm.dll")

type waveFormat struct {
	format, channels uint16
	rate, bytes      uint32
	align, bits, ext uint16
}

type waveHeader struct {
	data             *byte
	length, recorded uint32
	user             uintptr
	flags, loops     uint32
	next             uintptr
	reserved         uintptr
}

type waveBuffer struct {
	header   waveHeader
	pcm      [frameSamples * 2]byte
	prepared bool
}

type waveDevice struct {
	handle  uintptr
	input   bool
	buffers [3]waveBuffer
	pins    runtime.Pinner
	next    int
}

func waveCall(name string, args ...uintptr) error {
	code, _, _ := winmm.NewProc(name).Call(args...)
	if code != 0 {
		return fmt.Errorf("%s: audio error %d (check default devices and microphone permissions)", name, code)
	}
	return nil
}

func (d *waveDevice) function(suffix string) string {
	if d.input {
		return "waveIn" + suffix
	}
	return "waveOut" + suffix
}

func (d *waveDevice) submit(b *waveBuffer) error {
	name := "waveOutWrite"
	if d.input {
		name = "waveInAddBuffer"
	}
	return waveCall(name, d.handle, uintptr(unsafe.Pointer(&b.header)), unsafe.Sizeof(b.header))
}

func openWave(input bool, event uintptr) (*waveDevice, error) {
	d := &waveDevice{input: input}
	format := waveFormat{format: 1, channels: 1, rate: 8000, bytes: 16000, align: 2, bits: 16}
	if err := waveCall(d.function("Open"), uintptr(unsafe.Pointer(&d.handle)), uintptr(^uint32(0)), uintptr(unsafe.Pointer(&format)), event, 0, 0x50000); err != nil {
		return nil, err
	}
	for i := range d.buffers {
		b := &d.buffers[i]
		b.header.data, b.header.length = &b.pcm[0], uint32(len(b.pcm))
		// WinMM retains these addresses until Reset/UnprepareHeader completes.
		d.pins.Pin(&b.header)
		d.pins.Pin(&b.pcm[0])
		if err := waveCall(d.function("PrepareHeader"), d.handle, uintptr(unsafe.Pointer(&b.header)), unsafe.Sizeof(b.header)); err != nil {
			d.close()
			return nil, err
		}
		b.prepared = true
		if err := d.submit(b); err != nil {
			d.close()
			return nil, err
		}
	}
	if input {
		if err := waveCall("waveInStart", d.handle); err != nil {
			d.close()
			return nil, err
		}
	}
	return d, nil
}

func (d *waveDevice) close() {
	_ = waveCall(d.function("Reset"), d.handle)
	for i := range d.buffers {
		b := &d.buffers[i]
		if b.prepared {
			_ = waveCall(d.function("UnprepareHeader"), d.handle, uintptr(unsafe.Pointer(&b.header)), unsafe.Sizeof(b.header))
		}
	}
	_ = waveCall(d.function("Close"), d.handle)
	d.pins.Unpin()
}

func startAudio(bridge *audioBridge) (func(), error) {
	kernel := syscall.NewLazyDLL("kernel32.dll")
	createEvent := kernel.NewProc("CreateEventW")
	setEvent := kernel.NewProc("SetEvent")
	waitEvents := kernel.NewProc("WaitForMultipleObjects")
	var events [3]syscall.Handle
	closeEvents := func() {
		for _, event := range events {
			if event != 0 {
				_ = syscall.CloseHandle(event)
			}
		}
	}
	for i := range events {
		event, _, err := createEvent.Call(0, 0, 0, 0)
		if event == 0 {
			closeEvents()
			return nil, fmt.Errorf("create audio event: %w", err)
		}
		events[i] = syscall.Handle(event)
	}
	input, err := openWave(true, uintptr(events[0]))
	if err != nil {
		closeEvents()
		return nil, err
	}
	output, err := openWave(false, uintptr(events[1]))
	if err != nil {
		input.close()
		closeEvents()
		return nil, err
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer input.close()
		defer output.close()
		for {
			result, _, waitErr := waitEvents.Call(3, uintptr(unsafe.Pointer(&events[0])), 0, 0xffffffff)
			if result == 2 {
				return
			}
			if result > 2 {
				bridge.fail(fmt.Errorf("wait for audio: %w", waitErr))
				return
			}
			for _, device := range []*waveDevice{input, output} {
				for range device.buffers {
					b := &device.buffers[device.next]
					if b.header.flags&1 == 0 { // WHDR_DONE
						break
					}
					if device.input {
						n := min(int(b.header.recorded), len(b.pcm))
						bridge.process(nil, b.pcm[:n], 0)
					} else {
						bridge.process(b.pcm[:], nil, 0)
					}
					if err := device.submit(b); err != nil {
						bridge.fail(err)
						return
					}
					device.next = (device.next + 1) % len(device.buffers)
				}
			}
		}
	}()
	return func() {
		setEvent.Call(uintptr(events[2]))
		<-done
		closeEvents()
	}, nil
}
