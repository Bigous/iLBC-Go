package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"time"
)

type options struct {
	serve   string
	connect string
}

func parseOptions(args []string, output io.Writer) (options, error) {
	var opts options
	flags := flag.NewFlagSet("talk", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.StringVar(&opts.serve, "serve", "", "listen for one peer on this UDP port")
	flags.StringVar(&opts.connect, "connect", "", "call a peer at ip:port (IPv6: [ip]:port)")
	flags.Usage = func() {
		fmt.Fprintln(output, "Usage: talk --serve <port> | talk --connect <ip:port>")
		fmt.Fprintln(output, "Uses your default microphone and audio output. Press Ctrl+C to end the call.")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return opts, err
	}
	if flags.NArg() != 0 || (opts.serve == "") == (opts.connect == "") {
		return opts, errors.New("choose exactly one of --serve <port> or --connect <ip:port>")
	}
	port := opts.serve
	if opts.connect != "" {
		host, value, err := net.SplitHostPort(opts.connect)
		if err != nil || net.ParseIP(host) == nil {
			return opts, errors.New("--connect requires an IP address and port; use [IPv6]:port for IPv6")
		}
		port = value
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return opts, errors.New("UDP port must be between 1 and 65535")
	}
	return opts, nil
}

func run(ctx context.Context, opts options, output io.Writer) error {
	var remote, local *net.UDPAddr
	var err error
	if opts.connect != "" {
		remote, err = net.ResolveUDPAddr("udp", opts.connect)
	} else {
		local, err = net.ResolveUDPAddr("udp", ":"+opts.serve)
	}
	if err != nil {
		return err
	}
	network := "udp"
	if remote != nil && remote.IP.To4() != nil {
		network = "udp4"
	}
	conn, err := net.ListenUDP(network, local)
	if err != nil {
		return err
	}
	defer conn.Close()
	handshakeContext := ctx
	if remote != nil {
		var cancel context.CancelFunc
		handshakeContext, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		fmt.Fprintln(output, "Calling", remote)
	} else {
		fmt.Fprintln(output, "Listening on UDP", conn.LocalAddr())
	}
	p, err := handshake(handshakeContext, conn, remote)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	fmt.Fprintln(output, "Connected to", p.address, "— microphone active; Ctrl+C to stop.")
	return runCall(ctx, p, startAudio)
}

func main() {
	opts, err := parseOptions(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err == nil {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		err = run(ctx, opts, os.Stdout)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, errPeerLeft) {
		fmt.Println("Call ended.")
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "talk:", err)
		os.Exit(1)
	}
}
