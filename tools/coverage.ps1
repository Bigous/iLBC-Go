param(
    [switch]$OpenReport
)

$ErrorActionPreference = 'Stop'
$projectDir = Split-Path -Parent $PSScriptRoot
$reportDir = Join-Path $projectDir 'diagnostics\coverage'
$profilePath = Join-Path $reportDir 'coverage.out'
$htmlPath = Join-Path $reportDir 'coverage.html'
New-Item -ItemType Directory -Path $reportDir -Force | Out-Null

Push-Location $projectDir
try {
    # Disable cached test results so each invocation measures a fresh run.
    & go test './...' '-count=1' '-coverpkg=./...' "-coverprofile=$profilePath"
    if ($LASTEXITCODE -ne 0) { throw 'Coverage tests failed.' }

    & go tool cover "-func=$profilePath" |
        Tee-Object -FilePath (Join-Path $reportDir 'coverage.txt')
    if ($LASTEXITCODE -ne 0) { throw 'Coverage summary generation failed.' }

    & go tool cover "-html=$profilePath" "-o=$htmlPath"
    if ($LASTEXITCODE -ne 0) { throw 'HTML coverage report generation failed.' }

    Write-Host "HTML coverage report: $htmlPath"
    if ($OpenReport) { Invoke-Item -LiteralPath $htmlPath }
} finally {
    Pop-Location
}
