package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

type peer struct {
	conn    *net.UDPConn
	address *net.UDPAddr
	session uint64
}

func sameAddress(a, b *net.UDPAddr) bool {
	return a != nil && b != nil && a.Port == b.Port && a.Zone == b.Zone && a.IP.Equal(b.IP)
}

func (p *peer) send(message packet) error {
	var data [maxDatagram]byte
	message.session = p.session
	n := message.marshal(data[:])
	if err := p.conn.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		return err
	}
	_, err := p.conn.WriteToUDP(data[:n], p.address)
	return err
}

// Server accepts one peer for the lifetime of the call. Hello retries allow
// the client to recover a lost accept datagram without starting another call.
func handshake(ctx context.Context, conn *net.UDPConn, remote *net.UDPAddr) (*peer, error) {
	p := &peer{conn: conn, address: remote}
	if remote != nil {
		var id [8]byte
		if _, err := rand.Read(id[:]); err != nil {
			return nil, err
		}
		p.session = binary.BigEndian.Uint64(id[:]) | 1
	}
	// Read a complete UDP payload so oversized input is discarded by validation
	// rather than terminating a Windows socket with WSAEMSGSIZE.
	var data [65535]byte
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if remote != nil {
			if err := p.send(packet{kind: kindHello}); err != nil {
				return nil, err
			}
		}
		deadline := time.Now().Add(250 * time.Millisecond)
		if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
			deadline = d
		}
		if err := conn.SetReadDeadline(deadline); err != nil {
			return nil, err
		}
		n, from, err := conn.ReadFromUDP(data[:])
		if err != nil {
			if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
				continue
			}
			return nil, err
		}
		message, ok := parsePacket(data[:n])
		if !ok {
			continue
		}
		if remote == nil && message.kind == kindHello {
			p.address, p.session = from, message.session
			if err = p.send(packet{kind: kindAccept}); err != nil {
				return nil, err
			}
			return p, nil
		}
		if sameAddress(from, remote) && message.session == p.session && message.kind == kindAccept {
			return p, nil
		}
	}
}

func (p *peer) receive(ctx context.Context, incoming chan<- packet) error {
	var data [65535]byte
	lastAudio := time.Now()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Since(lastAudio) > 10*time.Second {
			return fmt.Errorf("peer timed out: no audio received for 10 seconds")
		}
		if err := p.conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond)); err != nil {
			return err
		}
		n, from, err := p.conn.ReadFromUDP(data[:])
		if err != nil {
			if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
				continue
			}
			return err
		}
		message, ok := parsePacket(data[:n])
		if !ok || !sameAddress(from, p.address) || message.session != p.session {
			continue
		}
		switch message.kind {
		case kindHello:
			if err := p.send(packet{kind: kindAccept}); err != nil {
				return err
			}
		case kindBye:
			return errPeerLeft
		case kindAudio:
			lastAudio = time.Now()
			select {
			case incoming <- message:
			default: // Bounded queue, no unbounded memory or network backpressure.
			}
		}
	}
}
