package main

import (
	"encoding/binary"
	"errors"
)

const (
	frameSamples = 160
	frameBytes   = 38
	headerBytes  = 20
	maxDatagram  = headerBytes + frameBytes
	kindHello    = 1
	kindAccept   = 2
	kindAudio    = 3
	kindBye      = 4
)

// The example uses its own fixed 20 ms wire format, not RTP.
type packet struct {
	kind    byte
	session uint64
	seq     uint32
	audio   [frameBytes]byte
}

func (p packet) marshal(dst []byte) int {
	clear(dst[:headerBytes])
	copy(dst, "ILBT")
	dst[4], dst[5] = 1, p.kind
	binary.BigEndian.PutUint64(dst[8:16], p.session)
	binary.BigEndian.PutUint32(dst[16:20], p.seq)
	if p.kind == kindAudio {
		copy(dst[headerBytes:], p.audio[:])
		return maxDatagram
	}
	return headerBytes
}

func parsePacket(data []byte) (packet, bool) {
	var p packet
	if len(data) < headerBytes || string(data[:4]) != "ILBT" || data[4] != 1 || data[6] != 0 || data[7] != 0 {
		return p, false
	}
	p.kind = data[5]
	if p.kind < kindHello || p.kind > kindBye {
		return p, false
	}
	expected := headerBytes
	if p.kind == kindAudio {
		expected = maxDatagram
	}
	if len(data) != expected {
		return p, false
	}
	p.session = binary.BigEndian.Uint64(data[8:16])
	p.seq = binary.BigEndian.Uint32(data[16:20])
	if p.session == 0 {
		return packet{}, false
	}
	copy(p.audio[:], data[headerBytes:])
	return p, true
}

var errPeerLeft = errors.New("peer disconnected")

// A bounded reorder window; all access belongs to the playout goroutine.
type jitterBuffer struct {
	slots   [64]packet
	valid   [64]bool
	next    uint32
	started bool
	wait    int
}

func (j *jitterBuffer) insert(p packet) {
	if !j.started {
		j.next, j.started, j.wait = p.seq, true, 3
	}
	delta := int32(p.seq - j.next)
	if delta < 0 {
		return // Already played or too late.
	}
	if delta >= int32(len(j.slots)) {
		// Bound latency after a pause or sender queue overflow.
		clear(j.valid[:])
		j.next, j.wait = p.seq, 3
	}
	i := p.seq % uint32(len(j.slots))
	if !j.valid[i] {
		j.slots[i], j.valid[i] = p, true
	}
}

func (j *jitterBuffer) take() (p packet, received, active bool) {
	if !j.started {
		return
	}
	if j.wait > 0 {
		j.wait--
		return
	}
	i := j.next % uint32(len(j.slots))
	p = j.slots[i]
	received = j.valid[i] && p.seq == j.next
	j.valid[i] = false
	j.next++
	return p, received, true
}
