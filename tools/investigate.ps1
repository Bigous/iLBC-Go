param(
    [ValidateRange(1, 3600)]
    [int]$FuzzSeconds = 60
)

$ErrorActionPreference = 'Stop'
$projectDir = Split-Path -Parent $PSScriptRoot
$reportDir = Join-Path $projectDir 'diagnostics'
New-Item -ItemType Directory -Path $reportDir -Force | Out-Null
Push-Location $projectDir
try {
    & go version | Tee-Object -FilePath (Join-Path $reportDir 'environment.txt')
    if ($LASTEXITCODE -ne 0) { throw 'go version failed' }

    # All commands respect the machine's existing application control policy.
    & go test -c -o (Join-Path $reportDir 'ilbc.diagnostics.test.exe')
    if ($LASTEXITCODE -ne 0) { throw 'Diagnostic build failed' }
    $testBinary = Join-Path $reportDir 'ilbc.diagnostics.test.exe'

    & $testBinary '-test.run=.' '-test.v' 2>&1 |
        Tee-Object -FilePath (Join-Path $reportDir 'tests.txt')
    if ($LASTEXITCODE -ne 0) { throw 'Tests failed; inspect diagnostics/tests.txt' }

    & $testBinary '-test.run=^$' '-test.bench=BenchmarkConcealScenarios' '-test.benchmem' '-test.benchtime=500ms' '-test.count=3' 2>&1 |
        Tee-Object -FilePath (Join-Path $reportDir 'conceal-scenarios.txt')
    if ($LASTEXITCODE -ne 0) { throw 'Diagnostic benchmarks failed' }

    # Selecting only BenchmarkCodec/Conceal would skip its preceding Decode
    # benchmark, leaving an unseeded decoder and changing the workload.
    & $testBinary '-test.run=^$' '-test.bench=BenchmarkConcealScenarios/mode.*/enhancetrue/losses512/flushfalse$' '-test.benchmem' '-test.benchtime=3s' "-test.cpuprofile=$reportDir\conceal.cpu.prof" 2>&1 |
        Tee-Object -FilePath (Join-Path $reportDir 'conceal-benchmark.txt')
    if ($LASTEXITCODE -ne 0) { throw 'CPU profiling failed' }

    & go tool pprof -top $testBinary (Join-Path $reportDir 'conceal.cpu.prof') 2>&1 |
        Tee-Object -FilePath (Join-Path $reportDir 'conceal-cpu.txt')
    if ($LASTEXITCODE -ne 0) { throw 'CPU profile analysis failed' }

    & go test -run '^$' -fuzz '^FuzzDecode$' -fuzztime "${FuzzSeconds}s" -parallel 4 2>&1 |
        Tee-Object -FilePath (Join-Path $reportDir 'fuzz.txt')
    if ($LASTEXITCODE -ne 0) { throw 'Fuzzing failed; inspect diagnostics/fuzz.txt and testdata/fuzz' }
} finally {
    Pop-Location
}
