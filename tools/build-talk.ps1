param(
    [ValidateSet('windows', 'linux', 'darwin')]
    [string]$TargetOS = 'windows',
    [ValidateSet('amd64', 'arm64')]
    [string]$TargetArch = 'amd64'
)

$ErrorActionPreference = 'Stop'
$projectDir = Split-Path -Parent $PSScriptRoot
$outputDir = Join-Path $projectDir 'bin'
New-Item -ItemType Directory -Force $outputDir | Out-Null
$name = "talk-$TargetOS-$TargetArch"
if ($TargetOS -eq 'windows') { $name += '.exe' }
if ($TargetOS -eq 'windows' -and $TargetArch -eq 'amd64') { $name = 'talk.exe' }
$savedCGO = $env:CGO_ENABLED
$savedOS = $env:GOOS
$savedArch = $env:GOARCH
try {
    $env:CGO_ENABLED = '0'
    $env:GOOS = $TargetOS
    $env:GOARCH = $TargetArch
    & go -C (Join-Path $projectDir 'cmd\talk') build -trimpath -o (Join-Path $outputDir $name) .
    if ($LASTEXITCODE -ne 0) { throw 'talk build failed.' }
    Write-Host "Built: $(Join-Path $outputDir $name) (CGO_ENABLED=0)"
} finally {
    $env:CGO_ENABLED = $savedCGO
    $env:GOOS = $savedOS
    $env:GOARCH = $savedArch
}
