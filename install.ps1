$ErrorActionPreference = "Stop"

$Repo = "TechXploreLabs/seristack"
$BinaryName = "seristack"

# ------------------------------------------------------------
# Detect Windows architecture
# ------------------------------------------------------------
$Arch = if ([Environment]::Is64BitOperatingSystem) {
    "amd64"
} else {
    "386"
}

# ------------------------------------------------------------
# Get latest GitHub release
# ------------------------------------------------------------
Write-Host "Checking latest seristack release..."

$ReleaseInfo = Invoke-RestMethod `
    -Uri "https://api.github.com/repos/$Repo/releases/latest"

$Tag = $ReleaseInfo.tag_name
$Version = $Tag.TrimStart("v")

# ------------------------------------------------------------
# Build release asset name
# ------------------------------------------------------------
$ArchiveName = "${BinaryName}_${Version}_windows_${Arch}.tar.gz"

$Asset = $ReleaseInfo.assets |
    Where-Object { $_.name -eq $ArchiveName } |
    Select-Object -First 1

if (-not $Asset) {
    Write-Host ""
    Write-Host "ERROR: Release asset not found." -ForegroundColor Red
    Write-Host ""
    Write-Host "Expected:"
    Write-Host "  $ArchiveName"
    Write-Host ""
    Write-Host "Available assets:"

    foreach ($ReleaseAsset in $ReleaseInfo.assets) {
        Write-Host "  $($ReleaseAsset.name)"
    }

    exit 1
}

$DownloadUrl = $Asset.browser_download_url

# ------------------------------------------------------------
# Temporary directory
# ------------------------------------------------------------
$TempFolder = Join-Path $env:TEMP "seristack_install"

if (Test-Path $TempFolder) {
    Remove-Item -Recurse -Force $TempFolder
}

New-Item `
    -ItemType Directory `
    -Force `
    -Path $TempFolder | Out-Null

$ArchivePath = Join-Path $TempFolder $ArchiveName

# ------------------------------------------------------------
# Download
# ------------------------------------------------------------
Write-Host ""
Write-Host "Downloading ${BinaryName} ${Tag}..."
Write-Host "Asset: $ArchiveName"

Invoke-WebRequest `
    -Uri $DownloadUrl `
    -OutFile $ArchivePath

# ------------------------------------------------------------
# Extract
# ------------------------------------------------------------
Write-Host "Extracting..."

tar -xzf $ArchivePath -C $TempFolder

# ------------------------------------------------------------
# Locate executable
# ------------------------------------------------------------
$ExeSource = Join-Path $TempFolder "${BinaryName}.exe"

if (-not (Test-Path $ExeSource)) {
    Write-Host ""
    Write-Host "ERROR: $BinaryName.exe was not found in the archive." -ForegroundColor Red
    Write-Host ""
    Write-Host "Archive contents:"

    Get-ChildItem -Recurse $TempFolder |
        ForEach-Object {
            Write-Host "  $($_.FullName)"
        }

    Remove-Item -Recurse -Force $TempFolder
    exit 1
}

# ------------------------------------------------------------
# Install destination
# ------------------------------------------------------------
$InstallDir = Join-Path `
    $env:LOCALAPPDATA `
    "Programs\seristack"

New-Item `
    -ItemType Directory `
    -Force `
    -Path $InstallDir | Out-Null

$ExeDestination = Join-Path `
    $InstallDir `
    "${BinaryName}.exe"

# ------------------------------------------------------------
# Install binary
# ------------------------------------------------------------
Write-Host "Installing to:"
Write-Host "  $InstallDir"

Move-Item `
    -Path $ExeSource `
    -Destination $ExeDestination `
    -Force

# ------------------------------------------------------------
# Add installation directory to User PATH
# ------------------------------------------------------------
$UserPath = [Environment]::GetEnvironmentVariable(
    "Path",
    "User"
)

$PathEntries = if ([string]::IsNullOrWhiteSpace($UserPath)) {
    @()
} else {
    $UserPath -split ";"
}

if ($PathEntries -notcontains $InstallDir) {

    $NewUserPath = if ([string]::IsNullOrWhiteSpace($UserPath)) {
        $InstallDir
    } else {
        "$UserPath;$InstallDir"
    }

    [Environment]::SetEnvironmentVariable(
        "Path",
        $NewUserPath,
        "User"
    )

    # Update PATH for the current PowerShell process
    if ($env:Path -notlike "*$InstallDir*") {
        $env:Path += ";$InstallDir"
    }

    Write-Host "Added $InstallDir to User PATH."
}

# ------------------------------------------------------------
# Cleanup
# ------------------------------------------------------------
Remove-Item `
    -Recurse `
    -Force `
    $TempFolder

# ------------------------------------------------------------
# Verify installation
# ------------------------------------------------------------
Write-Host ""
Write-Host "Installation complete!" -ForegroundColor Green
Write-Host "Version: $Tag"
Write-Host "Installed: $ExeDestination"
Write-Host ""

Write-Host "Testing seristack..."
& $ExeDestination --help

Write-Host ""
Write-Host "You may need to restart your terminal before running:"
Write-Host "  seristack"
