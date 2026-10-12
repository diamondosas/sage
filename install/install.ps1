$ErrorActionPreference = "Stop"

# --- Variables (Edit these if needed) ---
# Repository to download releases from (format: owner/repo)
# CHANGE HERE: Replace with your GitHub repository owner/name
$Repo = if ($env:SAGE_REPO) { $env:SAGE_REPO } else { "your-org/sage" }
# Target installation directory (change to custom path if desired)
# CHANGE HERE: Set custom install directory via env:INSTALL_DIR
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { "$env:LOCALAPPDATA\sage\bin" }
# Desired release tag ('latest' or specific version like 'v1.0.0')
$Tag = if ($env:SAGE_TAG) { $env:SAGE_TAG } else { "latest" }

# Detect Architecture matching GitHub Actions workflow matrix
$Arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }

# Binary name matching .github/workflows/windows.yml output
$Bin = "sage-windows-$Arch.exe"
$Dest = Join-Path $InstallDir "sage.exe"

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# 1. Use local build if present
if (Test-Path ".\dist\$Bin") {
    Copy-Item ".\dist\$Bin" $Dest -Force
} elseif (Test-Path ".\bin\sage.exe") {
    Copy-Item ".\bin\sage.exe" $Dest -Force
# 2. Or compile locally if Go compiler is present
} elseif ((Get-Command "go" -ErrorAction SilentlyContinue) -and (Test-Path ".\cmd\sage")) {
    Write-Host "Compiling Sage with local Go compiler..."
    go build -o $Dest .\cmd\sage
# 3. Otherwise download prebuilt release from GitHub (no Go required!)
} else {
    $Url = "https://github.com/$Repo/releases/download/$Tag/$Bin"
    Write-Host "Downloading prebuilt $Bin (no Go required)..."
    Invoke-WebRequest -Uri $Url -OutFile $Dest -UseBasicParsing
}

# Add InstallDir to User PATH if not already present
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -split ";" -notcontains $InstallDir) {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
    Write-Host "Added $InstallDir to User PATH."
}

Write-Host "Sage installed successfully! Run 'sage --help' to get started."
