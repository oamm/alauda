[CmdletBinding()]
param(
    [string]$Address = "127.0.0.1",
    [int]$Port = 9700,
    [string]$StoragePath = ".artifacts\dev-registry.db",
    [string]$DatabaseProvider = "postgres",
    [switch]$SkipUiBuild,
    [switch]$OpenBrowser,
    [switch]$ResetDevPassword
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

if ($DatabaseProvider -notin @("postgres", "sqlite")) { throw "Unsupported database provider '$DatabaseProvider'. Supported values: postgres, sqlite." }

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go is not available on PATH. Install Go 1.21+ or add it to PATH, then run this script again."
}

if ($DatabaseProvider -eq "postgres") {
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { throw "Docker is required for PostgreSQL development. Install Docker Desktop or use -DatabaseProvider sqlite." }
    docker compose version | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Docker Compose is unavailable. Install the Docker Compose plugin or use -DatabaseProvider sqlite." }
    $dockerEndpoint = docker context inspect --format '{{.Endpoints.docker.Host}}'
    if ($LASTEXITCODE -ne 0 -or -not $dockerEndpoint) { throw "Could not determine the active Docker context endpoint." }
    $postgresHost = $env:ALAUDA_POSTGRES_HOST
    if (-not $postgresHost) {
        $postgresHost = "localhost"
        if ($dockerEndpoint -match '^(tcp|ssh|http|https)://') {
            $postgresHost = ([uri]$dockerEndpoint.Trim()).Host
        }
    }
    $postgresPort = $env:ALAUDA_POSTGRES_PORT
    if (-not $postgresPort) {
        $postgresPort = if ($dockerEndpoint -match '^(tcp|ssh|http|https)://') { "5433" } else { "5432" }
        if ($postgresHost -in @("localhost", "127.0.0.1", "::1")) {
            $probe = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 5432)
            try { $probe.Start() } catch { $postgresPort = "5433" } finally { $probe.Stop() }
        }
        $env:ALAUDA_POSTGRES_PORT = $postgresPort
    }
    docker compose up -d --wait --wait-timeout 60 postgres
    if ($LASTEXITCODE -ne 0) { throw "PostgreSQL did not become healthy within 60 seconds." }
    $hostDatabaseReady = $false
    for ($attempt = 0; $attempt -lt 20; $attempt++) {
        $client = [System.Net.Sockets.TcpClient]::new()
        try {
            $connect = $client.BeginConnect($postgresHost, [int]$postgresPort, $null, $null)
            if ($connect.AsyncWaitHandle.WaitOne(500) -and $client.Connected) {
                $client.EndConnect($connect)
                $hostDatabaseReady = $true
                break
            }
        } catch {
        } finally {
            $client.Dispose()
        }
        Start-Sleep -Milliseconds 250
    }
    if (-not $hostDatabaseReady) {
        throw "PostgreSQL is healthy inside Docker, but Alauda cannot connect to ${postgresHost}:$postgresPort. Verify that the active Docker context publishes port 5432 on a host reachable from this machine, or set ALAUDA_POSTGRES_HOST and ALAUDA_POSTGRES_PORT."
    }
    $env:ALAUDA_STORAGE_PROVIDER = "postgres"
    $urlHost = if ($postgresHost.Contains(":")) { "[$postgresHost]" } else { $postgresHost }
    $env:ALAUDA_DATABASE_URL = "postgres://registry:dev_password@${urlHost}:$postgresPort/registry?sslmode=disable"
} else {
    $StoragePath = ".artifacts\alauda-dev.db"
    $storageDirectory = Split-Path -Parent $StoragePath
    New-Item -ItemType Directory -Force -Path $storageDirectory | Out-Null
    $env:ALAUDA_STORAGE_PROVIDER = "sqlite"
    $env:ALAUDA_DATABASE_URL = $StoragePath
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

$bootstrapCredentialPath = [System.IO.Path]::GetFullPath("$StoragePath.bootstrap-credential")
$bootstrapCredentialAlreadyExists = Test-Path -LiteralPath $bootstrapCredentialPath
if ($ResetDevPassword -and $bootstrapCredentialAlreadyExists) {
    Remove-Item -LiteralPath $bootstrapCredentialPath -Force
}

$env:REGISTRY_ADDRESS = $Address
$env:REGISTRY_PORT = "$Port"
$env:REGISTRY_STORAGE_PATH = $StoragePath
$env:REGISTRY_DEV_MODE = "true"
$env:REGISTRY_AUTH_ENABLED = "true"
$env:REGISTRY_BOOTSTRAP_CREDENTIAL_PATH = $bootstrapCredentialPath
$env:REGISTRY_RESET_DEV_BOOTSTRAP = if ($ResetDevPassword) { "true" } else { "false" }
$env:REGISTRY_TELEMETRY_ENABLED = "false"
$env:REGISTRY_LOG_LEVEL = "debug"

Write-Host "Applying database migrations ($DatabaseProvider)..."
& go run .\cmd\registry migrate up
if ($LASTEXITCODE -ne 0) { throw "Database migrations failed." }

$url = "http://$Address`:$Port"

Write-Host "Starting Service Registry in development mode..."
Write-Host "UI:      $url"
Write-Host "Health:  $url/healthz"
Write-Host "DB:      $DatabaseProvider"
Write-Host ""
$goCommand = Get-Command go -ErrorAction Stop
$registryProcess = Start-Process -FilePath $goCommand.Source -ArgumentList @("run", ".\cmd\registry", "--dev") -NoNewWindow -PassThru

try {
    $credentialNeedsWait = $ResetDevPassword -or -not $bootstrapCredentialAlreadyExists
    if ($credentialNeedsWait) {
        $credentialReady = $false
        for ($attempt = 0; $attempt -lt 60; $attempt++) {
            if ($registryProcess.HasExited) {
                throw "Registry stopped before bootstrap completed (exit code $($registryProcess.ExitCode))."
            }
            if (Test-Path -LiteralPath $bootstrapCredentialPath) {
                $credentialReady = $true
                break
            }
            Start-Sleep -Milliseconds 500
        }
        if (-not $credentialReady) {
            throw "Timed out waiting for bootstrap credential file: $bootstrapCredentialPath"
        }
    }

    if ($credentialNeedsWait) {
        $credentialLines = Get-Content -LiteralPath $bootstrapCredentialPath
        if ($credentialLines.Count -lt 2 -or $credentialLines[0] -notmatch '^username: ') {
            throw "Bootstrap credential file has an invalid format: $bootstrapCredentialPath"
        }
        $bootstrapUsername = $credentialLines[0].Substring("username: ".Length)
        $bootstrapPassword = $credentialLines[1]
        Write-Host "Development bootstrap credential ready." -ForegroundColor Green
        Write-Host "Username: $bootstrapUsername" -ForegroundColor Yellow
        Write-Host "Temporary password: $bootstrapPassword" -ForegroundColor Yellow
        Write-Host "Credential file: $bootstrapCredentialPath"
        Write-Host "Change this password after signing in." -ForegroundColor Yellow
        Write-Host ""
    } else {
        Write-Host "Using the existing development credential. Pass -ResetDevPassword to rotate it." -ForegroundColor Cyan
        Write-Host ""
    }

    Write-Host "Press Ctrl+C to stop."
    Write-Host ""
    if ($OpenBrowser) {
        Start-Process $url
    }
    Wait-Process -Id $registryProcess.Id
}
finally {
    if (-not $registryProcess.HasExited) {
        Stop-Process -Id $registryProcess.Id -Force
    }
}
