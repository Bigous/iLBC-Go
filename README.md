# iLBC in Go

A mono, 8 kHz, `int16` PCM encoder and decoder based on
[RFC 3951](https://www.rfc-editor.org/rfc/rfc3951.html). Requires Go 1.22 or later,
with no external dependencies, CGo, or `unsafe`.

| Mode | Samples/frame | Bytes/frame | Bit rate |
|---|---:|---:|---:|
| `Mode20` | 160 | 38 | 15.2 kbit/s |
| `Mode30` | 240 | 50 | 13.33 kbit/s |

```go
encoder, err := ilbc.NewEncoder(ilbc.Mode20)
if err != nil { return err }
decoder, err := ilbc.NewDecoder(ilbc.Mode20, true)
if err != nil { return err }

pcm := make([]int16, encoder.FrameSamples()) // Fill with 8 kHz audio.
packet := make([]byte, encoder.FrameBytes())
output := make([]int16, decoder.FrameSamples())

if err := encoder.Encode(packet, pcm); err != nil { return err }
if err := decoder.Decode(output, packet); err != nil { return err }
// When a packet is missing:
if err := decoder.Conceal(output); err != nil { return err }
```

The local module is named `ilbc`. To use it in another project, add
`require ilbc v0.0.0` and `replace ilbc => /path/to/ilbc` to `go.mod`,
then import `"ilbc"`. Replace the module path with your repository's public
address before publishing the library.

Reuse the buffers and keep one instance per stream. Instances retain history
and must not be shared concurrently between goroutines. Independent instances
can run in parallel. `Reset` clears stream history while retaining the
configuration. Initialize instances with `NewEncoder` or `NewDecoder`;
their zero values are not ready for use.

Each method processes one frame per call. WAV containers, RTP, resampling,
and automatic mode detection are outside the library's scope. Optional
enhancement adds a delay of 40 samples in 20 ms mode and 80 samples in 30 ms
mode. Packets with a loss marker or an invalid start index are handled by
packet loss concealment (PLC). Size validation errors leave stream state unchanged.

## Performance

Temporary buffers have fixed sizes and reside on the stack; output buffers
belong to the caller. Benchmarks measure **0 B/op and 0 allocs/op** for Encode,
Decode, and Conceal in both modes. Persistent state is allocated at construction.
The audio processing path uses no locks, reflection, or interfaces.

The codec core uses `float32`, preserves the reference implementation's relevant
rounding behavior, and uses `copy`/`clear` for buffer operations. The small private
`span` type represents filter history with offsets while retaining Go's bounds
checks. Explicit conversions on products prevent FMA fusion from changing the
rounding order. Tables are private and read-only during processing.

At frame boundaries, the decoder clears nonzero subnormal values (magnitudes
below 2^-126) from synthesis and high-pass filter history. This prevents tiny
silent tails from making repeated packet loss expensive, without changing CPU
floating-point modes. Codebook dot products check vector lengths once before
the inner loop, preserving sequential accumulation and product rounding.

Core variable declarations precede loops, following the reference's organization.
In Go, `:=` inside a loop does not imply heap allocation: escape analysis and
value lifetimes determine allocation behavior. Allocation tests verify the
result rather than inferring it from syntax.

## Testing and validation status

On Windows, run `.\tools\coverage.ps1` to execute fresh coverage tests and
generate text and HTML reports under `diagnostics/coverage/`. Add `-OpenReport`
to open the HTML report in your default browser.

```text
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
go test -run=^$ -bench=. -benchmem
go test -fuzz=FuzzDecode -fuzztime=60s
```

**The current suite passes with 100.0% statement coverage** on Windows/amd64
with Go 1.26.4. Reference comparisons report identical encoded bytes and zero
PCM differences in both modes, with and without enhancement. Tests and
benchmarks report **0 B/op and 0 allocs/op**. No production files are excluded
from coverage. Static analysis with `go vet ./...` also passes.

The final 60-second fuzzing campaign passed with 807,108 executions and four
workers. Controlled local benchmarks measured approximately 4x faster long-loss
Conceal and 14-16% faster Encode. See [VALIDATION.md](VALIDATION.md) for the
measurements, validation scope, and remaining platform checks.

Reference tests use 320 frames per mode, covering silence, impulses, sine waves,
noise, PCM extremes, and isolated and consecutive packet losses. The test
requires identical encoded bytes and allows a maximum difference of one PCM unit
across platforms; the Windows tests reported zero differences. This
differential test corpus is not a complete conformance certification.

The original C sources, corrections, and test vectors are in `testdata/`.
`tools/generate_inputs.py` recreates the inputs, and `tools/generate_vectors.py`
independently compiles the C reference and regenerates the expected results.
Python and a C compiler are required only to regenerate these fixtures.

## Documented differences from the reference

The following corrections affect the decoder and PLC, not the frame format.
Section 4.5 of the RFC permits PLC variations without affecting interoperability.

1. **20 ms decoder without enhancement:** the reference calculates correlation
   using the end of a 240-sample buffer even though only 160 samples are
   initialized. The window now stays within the frame: 40 samples for 20 ms
   and 80 samples for 30 ms, preserving the search for periods of 20 to 119 samples.
2. **Consecutive packet losses:** attenuation thresholds are evaluated from
   largest to smallest. In the reference, the first `>320` condition makes the
   remaining thresholds unreachable. Fallback noise also respects attenuation,
   and the counter saturates at 9 frames, enough to reach the final threshold
   in either mode.
3. **Recovery after packet loss:** enhancement preserves the period calculated
   before recovery interpolation. The reference reuses `lag` at a different
   scale and can return twice its intended value, causing an access outside
   the history buffer during the next PLC call.
4. **Subnormal filter tails:** the Go decoder clears synthesis and high-pass
   subnormal history before processing each frame. The C oracle does not apply
   this cleanup. Internal floating-point states can therefore differ, while
   encoded bytes and decoded int16 PCM remain identical in the tested corpus.

Provably unreachable branches were also removed: `CB_RESRANGE == -1` when the
constant is 34, `i < lengthIn` in the extension starting at `lengthIn+2`, and
a 20 ms start index greater than 3 in a two-bit field. Gain clamping uses
`min`/`max` while preserving its limits.

See `testdata/reference/corrections.patch` and the attribution notices in `LICENSE`.
