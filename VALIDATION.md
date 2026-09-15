# Validation record

Validated locally on 2026-09-15 with Windows/amd64, Go 1.26.4, and an Intel
Core i9-14900HX. Reference fixtures use RFC 3951 C sources compiled with
MSVC /O2 /fp:precise and the documented decoder/PLC corrections.

## Correctness and coverage

- The final production changes pass the full unit suite and examples with
  **100.0% statement coverage**, without excluding production files.
- Reference comparisons pass for 20 ms and 30 ms frames, with enhancement on
  and off: encoded bytes match exactly, PCM max delta is zero, and there are
  no differing samples in the fixture corpus.
- Allocation tests and all six codec benchmarks report **0 B/op, 0 allocs/op**.
- `go vet ./...` passes.
- New regression tests cover positive and negative subnormal boundaries,
  preservation of normal values, signed zeros, infinities and NaN payloads,
  long-loss history, packet recovery, and unchanged state on validation errors.

The reference corpus contains 320 frames per mode and includes silence,
impulses, sine waves, noise, PCM extremes, and isolated and consecutive losses.
Tests permit one PCM unit of variation across platforms; the local result is
exact. Statement coverage does not establish complete branch coverage or
absence of bugs, and this corpus is not a conformance certification.

## Confirmed performance diagnosis

The user's investigation completed all diagnostic tests and a 60-second,
four-worker fuzzing campaign: **1,141,376 executions**, 97 new interesting
inputs, and PASS. Twenty valid reference packets supplement the three original
fuzz seeds to exercise received frames in both modes and enhancement settings.

After prolonged loss, the original decoder retained 10 subnormal synthesis
history values and four subnormal high-pass history values. Residual and
enhancement histories had none. Matched snapshots after 512 losses with
enhancement enabled measured approximately 195 versus 46 microseconds in 20 ms
mode and 296 versus 75 microseconds in 30 ms mode when these tails were cleared.
The CPU profile attributed 51.30% of samples directly to `syntFilter` and
25.82% to `hpOutput`. Together, these measurements confirmed the cause of the
long-loss slowdown.

The decoder now clears only nonzero subnormal synthesis and high-pass history
at frame boundaries, using integer bit tests. It preserves normal values and
signed zeros and does not change process or CPU floating-point settings.
Validation errors still leave decoder state unchanged. The C oracle has no
such cleanup, so internal floating-point history can differ; the tested int16
output remains identical.

The encoder's three codebook dot-product loops now check vector extents once
and use direct slice access. Sequential accumulation and explicit float32
product rounding preserve the reference arithmetic.

## Controlled before/after benchmarks

Medians of three paired baseline/final runs on the same machine and Go version,
with alternating execution order, `GOMAXPROCS=1`, and 300 ms per benchmark:

```powershell
$env:GOMAXPROCS = '1'
.\ilbc.test.exe '-test.run=^$' '-test.bench=^BenchmarkCodec$' '-test.benchmem' '-test.benchtime=300ms'
```

| Operation | Before (ns/op) | After (ns/op) | Change |
|---|---:|---:|---:|
| Encode20 | 129,070 | 110,926 | 14.1% less time |
| Decode20 | 52,437 | 51,826 | 1.2% less time |
| Conceal20 | 197,868 | 47,034 | 4.21x faster |
| Encode30 | 220,286 | 184,979 | 16.0% less time |
| Decode30 | 87,280 | 85,695 | 1.8% less time |
| Conceal30 | 301,907 | 76,865 | 3.93x faster |

Every measurement reported zero allocations. The small Decode differences
should be treated as measurement noise. These are local results, not portable
performance guarantees.

`BenchmarkCodec` enables enhancement and shares history between Decode and
Conceal. Its repeated Conceal loop mostly measures long-loss behavior. Running
only that sub-benchmark skips the preceding Decode workload and changes the
history. Diagnostic snapshot benchmarks include equal state-copy costs in both
variants and should not be directly compared with the evolving-stream table.
With the fix, the original/cleared diagnostic variants should converge because
production already clears the relevant histories.

## Fuzzing and reproduction

A separate 60-second campaign after the decoder fix passed with **1,028,445
executions**, four workers, and 10 new interesting inputs. This preceded the
encoder dot-product optimization.

The final revision passed another 60-second campaign with **807,108 executions**,
four workers, and all 144 cached baseline inputs; no new interesting inputs or
failures were reported.

From the repository root in PowerShell:

```powershell
go test ./... '-coverprofile=coverage.out'
go tool cover '-func=coverage.out'
go vet ./...
.\tools\investigate.ps1 -FuzzSeconds 60
```

The script builds and runs tests, repeats diagnostic benchmarks three times,
collects a CPU profile, and fuzzes with four workers. Logs go under the ignored
`diagnostics/` directory. Final optimization evidence is in
`diagnostics/optimization/`. Preserve any failing reproducer under
`src/testdata/fuzz`. A successful bounded campaign does not prove that no bugs exist.

## Remaining validation

Other operating systems, architectures, and Go versions have not been measured
in this local run. GitHub run 34972213321 passed on Ubuntu with Go 1.22 and stable on 2026-09-15, before the release metadata changes. Recheck performance on deployment hardware and
consider longer fuzzing campaigns before release.

Direct execution of the rebuilt repository coverage executable was blocked by Windows Application Control in this session. Validation through go test succeeded; no application control policy was changed.

## src layout and talk validation

The codec and reference fixtures now live under `src/`. The public package
import is `github.com/Bigous/iLBC-Go/src`; the root module path is unchanged.
The coverage script still reports 100.0% codec statement coverage after the move.

The separate `cmd/talk` module builds without CGo. GitHub run
[35007064822](https://github.com/Bigous/iLBC-Go/actions/runs/35007064822)
passed native builds and synthetic-audio tests on Windows, Linux, and macOS,
Go 1.22 compatibility on Ubuntu, and Linux race detection. Its 60-second
packet-parser fuzz campaign passed with 4,270,007 executions and seven new
interesting inputs. Local cross-builds cover amd64 and arm64 on all three OSes.
The Windows executable's `--help` command also ran successfully.

Talk tests cover bidirectional UDP transport through the real codec with
synthetic microphone/output buffers, malformed and oversized packets, peer
and session filtering, handshake retries, timeouts, packet loss/reordering,
sequence wraparound, bounded queues, and shutdown. These are separate from
the codec's coverage result. Native audio device access is not exercised by
CI, and a two-machine microphone/speaker listening test remains a manual check.
