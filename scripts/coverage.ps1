param(
    [double]$Threshold = 80.0,
    [string]$OutDir = ".artifacts"
)

$ErrorActionPreference = "Stop"

if (-not $env:CGO_ENABLED) {
    $env:CGO_ENABLED = "1"
}
if (-not $env:CC) {
    $env:CC = "C:\ProgramData\mingw64\mingw64\bin\gcc.exe"
}
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path (Get-Location) ".artifacts\gocache"
}
if (-not $env:GOMODCACHE) {
    $env:GOMODCACHE = Join-Path (Get-Location) ".artifacts\gomodcache"
}

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$allPackages = go list ./...
$testPackages = $allPackages | Where-Object {
    $_ -notmatch '/gen/' -and
    $_ -notmatch '/web/' -and
    $_ -notmatch '/internal/webstatic$' -and
    $_ -notmatch '/cmd/registry$'
}
$coverPackages = $testPackages | Where-Object {
    $_ -notmatch '/tests/integration$'
}

$profile = Join-Path $OutDir "coverage.out"
$summary = Join-Path $OutDir "coverage-summary.txt"
$html = Join-Path $OutDir "coverage.html"
$coverPkgArg = $coverPackages -join ","

go test "-coverpkg=$coverPkgArg" "-coverprofile=$profile" @testPackages
$coverageOutput = go tool cover "-func=$profile"
$coverageOutput | Tee-Object -FilePath $summary
go tool cover "-html=$profile" "-o=$html"

$totalLine = $coverageOutput | Where-Object { $_ -match '^total:\s+\(statements\)\s+([0-9.]+)%' }
if (-not $totalLine) {
    throw "Unable to parse total coverage from go tool cover output."
}

$actual = [double]($Matches[1])
if ($actual -lt $Threshold) {
    Write-Error ("Coverage {0:N1}% is below required threshold {1:N1}%." -f $actual, $Threshold)
}

Write-Host ("Coverage {0:N1}% meets threshold {1:N1}%." -f $actual, $Threshold)
Write-Host "Coverage summary: $summary"
Write-Host "Coverage HTML: $html"
