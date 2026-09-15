package main

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func localSocket(t *testing.T) *net.UDPConn {
	t.Helper()
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestHandshakeAndPeerFiltering(t *testing.T) {
	server, client, stranger := localSocket(t), localSocket(t), localSocket(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	serverPeer := make(chan *peer, 1)
	failures := make(chan error, 1)
	go func() {
		p, err := handshake(ctx, server, nil)
		if err != nil {
			failures <- err
		} else {
			serverPeer <- p
		}
	}()
	p, err := handshake(ctx, client, server.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	var s *peer
	select {
	case s = <-serverPeer:
	case err := <-failures:
		t.Fatal(err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if p.session != s.session || !sameAddress(s.address, client.LocalAddr().(*net.UDPAddr)) {
		t.Fatal("handshake selected incorrect peer")
	}
	incoming := make(chan packet, 4)
	go func() { failures <- s.receive(ctx, incoming) }()
	if _, err := stranger.WriteToUDP(make([]byte, 2048), server.LocalAddr().(*net.UDPAddr)); err != nil {
		t.Fatal(err)
	}
	spoof := &peer{conn: stranger, address: s.conn.LocalAddr().(*net.UDPAddr), session: s.session}
	if err := spoof.send(packet{kind: kindAudio, seq: 90}); err != nil {
		t.Fatal(err)
	}
	wrongSession := *p
	wrongSession.session++
	if err := wrongSession.send(packet{kind: kindAudio, seq: 91}); err != nil {
		t.Fatal(err)
	}
	// A lost handshake reply can be recovered while the server is in the call.
	if err := p.send(packet{kind: kindHello}); err != nil {
		t.Fatal(err)
	}
	var raw [maxDatagram]byte
	client.SetReadDeadline(time.Now().Add(time.Second))
	n, _, err := client.ReadFromUDP(raw[:])
	if err != nil {
		t.Fatal(err)
	}
	if ack, ok := parsePacket(raw[:n]); !ok || ack.kind != kindAccept {
		t.Fatal("hello retry not acknowledged")
	}
	if err := p.send(packet{kind: kindAudio, seq: 7}); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-incoming:
		if got.seq != 7 {
			t.Fatal("accepted another peer/session")
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	p.send(packet{kind: kindBye})
	select {
	case err := <-failures:
		if !errors.Is(err, errPeerLeft) {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("disconnect did not stop receiver")
	}
}

func TestHandshakeDeadline(t *testing.T) {
	for _, clientMode := range []bool{false, true} {
		conn, idle := localSocket(t), localSocket(t)
		var address *net.UDPAddr
		if clientMode {
			address = idle.LocalAddr().(*net.UDPAddr)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		_, err := handshake(ctx, conn, address)
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected deadline, got %v", err)
		}
	}
}

func TestBidirectionalCall(t *testing.T) {
	left, right := localSocket(t), localSocket(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var audible [2]atomic.Int64
	openFake := func(side int) func(*audioBridge) (func(), error) {
		return func(b *audioBridge) (func(), error) {
			stop, done := make(chan struct{}), make(chan struct{})
			go func() {
				defer close(done)
				tick := time.NewTicker(20 * time.Millisecond)
				defer tick.Stop()
				var input, output [frameSamples * 2]byte
				for i := 0; i < frameSamples; i++ {
					value := int16(10000 * math.Sin(2*math.Pi*float64(i)*float64(400+side*200)/8000))
					binary.LittleEndian.PutUint16(input[2*i:], uint16(value))
				}
				for {
					select {
					case <-stop:
						return
					case <-tick.C:
						b.process(output[:], input[:], frameSamples)
						for i := 0; i < len(output); i += 2 {
							value := int16(binary.LittleEndian.Uint16(output[i:]))
							if value > 100 || value < -100 {
								audible[side].Add(1)
							}
						}
					}
				}
			}()
			return func() { close(stop); <-done }, nil
		}
	}
	results := make(chan error, 2)
	go func() { results <- runCall(ctx, &peer{left, right.LocalAddr().(*net.UDPAddr), 1}, openFake(0)) }()
	go func() { results <- runCall(ctx, &peer{right, left.LocalAddr().(*net.UDPAddr), 1}, openFake(1)) }()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for audible[0].Load() < 1000 || audible[1].Load() < 1000 {
		select {
		case err := <-results:
			t.Fatalf("call stopped before bidirectional audio: %v", err)
		case <-ctx.Done():
			t.Fatal("bidirectional audio missing")
		case <-tick.C:
		}
	}
	cancel()
	for i := 0; i < 2; i++ {
		select {
		case <-results:
		case <-time.After(2 * time.Second):
			t.Fatal("call workers did not shut down")
		}
	}
}

func TestAudioOpenFailure(t *testing.T) {
	conn, remote := localSocket(t), localSocket(t)
	want := errors.New("no microphone")
	err := runCall(context.Background(), &peer{conn, remote.LocalAddr().(*net.UDPAddr), 1}, func(*audioBridge) (func(), error) { return nil, want })
	if !errors.Is(err, want) {
		t.Fatal(err)
	}
}
