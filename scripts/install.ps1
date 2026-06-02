#Requires -Version 5.1
<#
.SYNOPSIS
    GoMud installer for Windows.

.DESCRIPTION
    Installs Git and Go (if needed), clones GoMud, and builds the server binary.

.EXAMPLE
    One-liner from an elevated PowerShell prompt:
    irm https://raw.githubusercontent.com/GoMudEngine/GoMud/master/scripts/install.ps1 | iex

.NOTES
    Environment variables:
        GOMUD_DIR   Override the install directory (default: $HOME\GoMud)
#>

$ErrorActionPreference = 'Stop'

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

$GomudRepo    = 'https://github.com/GoMudEngine/GoMud.git'
$GomudDir     = if ($env:GOMUD_DIR) { $env:GOMUD_DIR } else { Join-Path $HOME 'GoMud' }
$GoInstallDir = 'C:\Go'
$GoDlApi      = 'https://go.dev/dl/?mode=json'

# Minimum Go version required (must match go.mod)
$MinGoMajor = 1
$MinGoMinor = 24

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

function Info  { param([string]$Msg) Write-Host "==> $Msg" }
function Fatal { param([string]$Msg) Write-Error "error: $Msg"; exit 1 }

function Test-CommandExists {
    param([string]$Name)
    $null -ne (Get-Command $Name -ErrorAction SilentlyContinue)
}

function Get-CurrentArch {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        'X64'   { return 'amd64' }
        'Arm64' { return 'arm64' }
        'X86'   { return '386' }
        default { Fatal "Unsupported architecture: $arch" }
    }
}

function Add-ToMachinePath {
    param([string]$Dir)
    $current = [System.Environment]::GetEnvironmentVariable('PATH', 'Machine')
    if ($current -notlike "*$Dir*") {
        [System.Environment]::SetEnvironmentVariable('PATH', "$current;$Dir", 'Machine')
        Info "Added $Dir to the system PATH."
    }
    # Also update the current session.
    if ($env:PATH -notlike "*$Dir*") {
        $env:PATH = "$env:PATH;$Dir"
    }
}

# ---------------------------------------------------------------------------
# Execution policy check
# ---------------------------------------------------------------------------

$policy = Get-ExecutionPolicy -Scope CurrentUser
if ($policy -eq 'Restricted') {
    Write-Warning @"
Your PowerShell execution policy is set to Restricted, which prevents scripts from running.
To allow this installer to run, execute the following command and then re-run the installer:

    Set-ExecutionPolicy RemoteSigned -Scope CurrentUser
"@
    exit 1
}

# ---------------------------------------------------------------------------
# Git installation
# ---------------------------------------------------------------------------

function Install-Git {
    Info 'git not found. Attempting to install...'

    if (Test-CommandExists 'winget') {
        winget install --id Git.Git --silent --accept-package-agreements --accept-source-agreements
        # Refresh PATH so git is visible in the current session.
        $env:PATH = [System.Environment]::GetEnvironmentVariable('PATH', 'Machine') + ';' +
                    [System.Environment]::GetEnvironmentVariable('PATH', 'User')
    } elseif (Test-CommandExists 'choco') {
        choco install git -y
        $env:PATH = [System.Environment]::GetEnvironmentVariable('PATH', 'Machine') + ';' +
                    [System.Environment]::GetEnvironmentVariable('PATH', 'User')
    } else {
        Fatal @"
Cannot install git automatically on this system.
Please install git manually from https://git-scm.com/download/win
then re-run this installer.
"@
    }

    if (-not (Test-CommandExists 'git')) {
        Fatal 'git installation appeared to succeed but git is still not in PATH. Please restart your shell and re-run.'
    }
}

# ---------------------------------------------------------------------------
# Go version check
# ---------------------------------------------------------------------------

function Test-GoVersionOk {
    if (-not (Test-CommandExists 'go')) { return $false }
    try {
        $verLine = & go version 2>&1
        if ($verLine -match 'go(\d+)\.(\d+)') {
            $major = [int]$Matches[1]
            $minor = [int]$Matches[2]
            if ($major -gt $MinGoMajor) { return $true }
            if ($major -eq $MinGoMajor -and $minor -ge $MinGoMinor) { return $true }
        }
    } catch { }
    return $false
}

# ---------------------------------------------------------------------------
# Go installation
# ---------------------------------------------------------------------------

function Install-Go {
    param([string]$Arch)

    Info 'Fetching latest stable Go release information...'

    $releases = Invoke-RestMethod -Uri $GoDlApi -UseBasicParsing

    $latest = $releases | Where-Object { $_.stable -eq $true } | Select-Object -First 1
    if (-not $latest) { Fatal "Could not determine the latest stable Go version." }

    $goVersion = $latest.version
    $archiveName = "$goVersion.windows-$Arch.zip"

    $fileEntry = $latest.files | Where-Object { $_.filename -eq $archiveName } | Select-Object -First 1
    if (-not $fileEntry) {
        Fatal "Could not find a download entry for $archiveName in the Go release API."
    }

    $dlUrl    = "https://dl.google.com/go/$archiveName"
    $sha256   = $fileEntry.sha256
    $tmpPath  = Join-Path $env:TEMP $archiveName

    Info "Downloading $goVersion (windows/$Arch)..."
    Invoke-WebRequest -Uri $dlUrl -OutFile $tmpPath -UseBasicParsing

    Info 'Verifying checksum...'
    $actual = (Get-FileHash -Algorithm SHA256 -Path $tmpPath).Hash.ToLower()
    if ($actual -ne $sha256) {
        Remove-Item $tmpPath -Force
        Fatal "SHA256 mismatch for $archiveName. Expected $sha256, got $actual."
    }

    Info "Installing Go to $GoInstallDir..."
    if (Test-Path $GoInstallDir) {
        Remove-Item $GoInstallDir -Recurse -Force
    }

    Expand-Archive -Path $tmpPath -DestinationPath 'C:\' -Force
    Remove-Item $tmpPath -Force

    Add-ToMachinePath "$GoInstallDir\bin"

    Info "$goVersion installed."
}

# ---------------------------------------------------------------------------
# GoMud clone / update
# ---------------------------------------------------------------------------

function Setup-GomudRepo {
    if (Test-Path (Join-Path $GomudDir '.git')) {
        Info "GoMud directory already exists at $GomudDir. Pulling latest changes..."
        & git -C $GomudDir pull
    } elseif (Test-Path $GomudDir) {
        Fatal "$GomudDir exists but is not a git repository. Remove or rename it, or set GOMUD_DIR to a different path, and re-run."
    } else {
        Info "Cloning GoMud into $GomudDir..."
        & git clone $GomudRepo $GomudDir
    }
}

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

function Build-Gomud {
    Info 'Building GoMud...'
    Push-Location $GomudDir
    try {
        & go generate
        $env:CGO_ENABLED = '0'
        & go build -trimpath -a -o go-mud-server.exe
    } finally {
        Pop-Location
    }
    Info "Build complete: $GomudDir\go-mud-server.exe"
}

# ---------------------------------------------------------------------------
# Next steps
# ---------------------------------------------------------------------------

function Print-NextSteps {
    Write-Host ''
    Write-Host '================================================================'
    Write-Host ' GoMud is ready!'
    Write-Host '================================================================'
    Write-Host ''
    Write-Host 'Before starting for the first time, set an admin password:'
    Write-Host ''
    Write-Host "  cd $GomudDir"
    Write-Host '  go run .\cmd\reset-admin-pw'
    Write-Host ''
    Write-Host 'Start the server:'
    Write-Host ''
    Write-Host "  cd $GomudDir"
    Write-Host '  .\go-mud-server.exe'
    Write-Host ''
    Write-Host 'Connect:'
    Write-Host '  Web client : http://localhost/webclient'
    Write-Host '  Web admin  : http://localhost/admin/'
    Write-Host '  Telnet     : localhost:33333'
    Write-Host ''
    Write-Host 'For the full developer workflow (make run, make test, etc.):'
    Write-Host '  https://github.com/GoMudEngine/GoMud#build-commands'
    Write-Host ''
    Write-Host 'NOTE: If "go" is not recognized in your current shell, open a new'
    Write-Host "      terminal window so the updated PATH takes effect."
    Write-Host ''
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

$arch = Get-CurrentArch

if (-not (Test-CommandExists 'git')) {
    Install-Git
}

if (Test-GoVersionOk) {
    $installedVer = (& go version) -replace '.*go(\d+\.\d+[\.\d]*).*', '$1'
    Info "Go $installedVer already installed and meets the minimum requirement."
} else {
    Install-Go -Arch $arch
}

Setup-GomudRepo
Build-Gomud
Print-NextSteps
