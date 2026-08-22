[CmdletBinding()]
param(
    [string]$Address = "127.0.0.1",
    [int]$Port = 9700,
    [string]$StoragePath = ".artifacts\dev-registry.db",
    [switch]$SkipUiBuild,
    [switch]$OpenBrowser
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go is not available on PATH. Install Go 1.21+ or add it to PATH, then run this script again."
}

if (-not $SkipUiBuild) {
    $npmCommand = if ($IsWindows -or $PSVersionTable.PSEdition -eq "Desktop") { "npm.cmd" } else { "npm" }
    if (-not (Get-Command $npmCommand -ErrorAction SilentlyContinue)) {
        throw "npm is not available on PATH. Install Node.js/npm or run with -SkipUiBuild."
    }

    $webRoot = Join-Path $repoRoot "web"
    $webDist = Join-Path $webRoot "dist"
    $embeddedDist = Join-Path $repoRoot "internal\webstatic\dist"

    Write-Host "Building UI..."
    if (-not (Test-Path (Join-Path $webRoot "node_modules"))) {
        Push-Location $webRoot
        try {
            & $npmCommand install
        }
        finally {
            Pop-Location
        }
    }

    Push-Location $webRoot
    try {
        & $npmCommand run build
    }
    finally {
        Pop-Location
    }

    $resolvedEmbeddedDist = [System.IO.Path]::GetFullPath($embeddedDist)
    $resolvedRepoRoot = [System.IO.Path]::GetFullPath($repoRoot)
    if (-not $resolvedEmbeddedDist.StartsWith($resolvedRepoRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to update UI assets outside the repository: $resolvedEmbeddedDist"
    }

    New-Item -ItemType Directory -Force -Path $embeddedDist | Out-Null
    Get-ChildItem -LiteralPath $embeddedDist -Force | Remove-Item -Recurse -Force
    Copy-Item -Path (Join-Path $webDist "*") -Destination $embeddedDist -Recurse -Force
    Write-Host "UI assets updated: internal\webstatic\dist"
    Write-Host ""
}

$storageDirectory = Split-Path -Parent $StoragePath
if ($storageDirectory) {
    New-Item -ItemType Directory -Force -Path $storageDirectory | Out-Null
}

$env:REGISTRY_ADDRESS = $Address
$env:REGISTRY_PORT = "$Port"
$env:REGISTRY_STORAGE_PATH = $StoragePath
$env:REGISTRY_DEV_MODE = "true"
$env:REGISTRY_AUTH_ENABLED = "false"
$env:REGISTRY_TELEMETRY_ENABLED = "false"
$env:REGISTRY_LOG_LEVEL = "debug"

$url = "http://$Address`:$Port"

Write-Host "Starting Service Registry in development mode..."
Write-Host "UI:      $url"
Write-Host "Health:  $url/healthz"
Write-Host "DB:      $StoragePath"
Write-Host ""
Write-Host "Press Ctrl+C to stop."
Write-Host ""

if ($OpenBrowser) {
    Start-Process $url
}

go run .\cmd\registry --dev
