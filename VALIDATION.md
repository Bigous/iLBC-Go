# Validation record

Environment: Windows/amd64, Go 1.26.4, Intel Core i9-14900HX.
Reference executable: RFC 3951 sources compiled with MSVC, `/O2 /fp:precise`.

## Completed

- Initial comparison: 320 frames for each mode and enhancement configuration,
  with identical encoded bytes. After the initial PLC and correlation window
  corrections, PCM output matched in all four configurations.
- Coverage of that revision: **96.5% of statements**.
- A subsequent test alternating packet losses and receptions detected a
  doubled pitch period in the enhancement stage. The correction was applied
  to both Go and the C reference, and expected vectors were regenerated using C.
- The final revision and all tests compile (`go test -c`).
- Final static analysis: `go vet ./...` completed without diagnostics.

## Pending due to an execution restriction

Windows Code Integrity blocked loading `ilbc.test.exe`
(event 3077: Enterprise signing requirements / code integrity policy).
An attempt outside the sandbox produced the same result. The policy was
neither changed nor bypassed.

- Run all tests against the final revision.
- Verify the enhancement correction against the regenerated vectors.
- Measure final coverage and cover any remaining statements to reach 100%.
- Run fuzzing and repeat benchmarks after the latest changes.
- Validate other architectures and operating systems.

## Initial benchmarks

These measurements predate the final revision. Go benchmarks used
`-benchtime=100ms`, with decoder enhancement enabled. Timings vary with CPU
frequency, system conditions, and audio content.

| Operation | 20 ms | 30 ms | Allocations |
|---|---:|---:|---:|
| Encode | 151 µs/frame | 267 µs/frame | 0 B/op, 0 allocs/op |
| Decode | 53 µs/frame | 87 µs/frame | 0 B/op, 0 allocs/op |
| Conceal | 48 µs/frame | 78 µs/frame | 0 B/op, 0 allocs/op |

These results provide a reproducible initial baseline; they do not establish
that performance is optimal. Allocation tests cover both frame durations and
both enhancement settings, but their final execution is pending.
