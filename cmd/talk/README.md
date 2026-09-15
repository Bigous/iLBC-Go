# talk

A bidirectional iLBC voice-call example for Windows, Linux, and macOS. It
captures the default microphone, encodes 20 ms frames using this repository's
Go codec, sends them over UDP, decodes received frames, and plays them through
the default audio output. Both peers speak and listen simultaneously.

## Build

Requires Go 1.22 or later. No CGo, C compiler, FFmpeg, or bundled native audio
library is required. From the repository root:

```powershell
.\tools\build-talk.ps1
```

Or use Go directly on any supported platform:

```sh
cd cmd/talk
CGO_ENABLED=0 go build -o talk .
```

In PowerShell, set `$env:CGO_ENABLED = '0'` before `go build -o talk.exe .`.
This example has its own `go.mod` and uses a local replacement for the parent
codec module; build it from a clone of the repository. Audio dependencies are
isolated from the dependency-free codec module.

## Call

On the listening machine:

```powershell
.\bin\talk.exe --serve 9000
```

On the other machine, substitute the listener's IP address:

```powershell
.\bin\talk.exe --connect 192.168.1.10:9000
```

On Linux/macOS, run `./talk` from the build directory with the same arguments.
For IPv6 use `--connect '[2001:db8::1]:9000'`. Press Ctrl+C to end the call.
The listener handles one peer and exits when the call ends; restart it for a
new call. The microphone starts only after a peer handshake succeeds.

Allow inbound UDP on the selected port on the listening machine. Across NAT,
use a VPN or configure port forwarding; there is no STUN/TURN or discovery.
Use headphones: the example has no acoustic echo cancellation and speakers
can feed back into the microphone. Audio is unencrypted and peers are not
authenticated, so use a trusted LAN or VPN. The first valid caller is accepted.

## Audio backends

| System | Backend | Runtime requirement |
|---|---|---|
| Windows | WinMM through Go system calls | Default microphone/output enabled |
| Linux | Pure-Go PulseAudio client | PulseAudio or PipeWire with pipewire-pulse |
| macOS | AudioQueue through purego | System AudioToolbox; terminal microphone permission |

The backends request mono signed 16-bit PCM at 8 kHz. System audio services
perform any device-rate conversion; the iLBC codec itself does not resample.
On Linux, run within your desktop audio session. `PULSE_SERVER` can select a
non-default PulseAudio endpoint. ALSA-only systems without a PulseAudio-compatible
service are not supported. On macOS, launch from Terminal and grant microphone
access in System Settings. Select devices in the OS before starting a call;
automatic device switching is not implemented.

## Transport and tests

Each audio datagram contains a 20-byte versioned header and one 38-byte iLBC
frame. Header fields are magic `ILBT`, version 1, message kind, two reserved
zero bytes, a 64-bit call ID, and a 32-bit sequence number; integers use network
byte order. Hello/accept/bye packets contain only the header. This is an example
protocol, not RTP or a production telephony protocol.

The receiver checks the peer address, call ID, and exact packet length. A
bounded reorder window starts with three frames of buffering. Missing frames
use iLBC concealment, late/duplicate packets are discarded, and queues remain
bounded during stalls. Calls time out after ten seconds without received audio.
There is no silence suppression. Jitter buffering, device queues, and decoder
enhancement add latency beyond the 20 ms frame duration.

```sh
go test ./...
go test -race ./...  # Requires the race detector's C toolchain.
go test -run '^$' -fuzz FuzzPacket -fuzztime 60s
```

Tests cover packet validation, reordering/loss/wraparound, buffer backpressure,
handshakes, peer filtering, shutdown, and bidirectional UDP calls using synthetic
audio through the real codec. They do not require audio hardware. The library's
100% statement coverage is measured separately by `tools/coverage.ps1`; it is
not a claim of 100% coverage of these platform audio backends.

Hardware capture/playback quality must also be checked with two real machines.
See the repository's LICENSE, LICENSE-RFC3951, NOTICE, and this directory's
THIRD_PARTY_NOTICES for applicable notices.
