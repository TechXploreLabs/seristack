$ErrorActionPreference = "Stop"

$Repo = "TechXploreLabs/seristack"
$BinaryName = "seristack"

# Detect OS architecture
$Arch = if ([Environment]::Is64BitOperatingSystem) {
    "amd64"
} else {
    "386"
}

# Get latest release
$ReleaseInfo = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"

$Tag = $ReleaseInfo.tag_name
$Version = $Tag.TrimStart("v")

# Expected asset name
$ExpectedAssetName = "${BinaryName}_${Version}_windows_${Arch}.zip"

# Find the asset from the release
$Asset = $ReleaseInfo.assets |
    Where-Object { $_.name -eq $ExpectedAssetName } |
    Select-Object -First 1

if (-not $Asset) {
    Write-Host ""
    Write-Host "ERROR: Could not find Windows $Arch release asset." -ForegroundColor Red
    Write-Host ""
    Write-Host "Expected:"
    Write-Host "  $ExpectedAssetName"
    Write-Host ""
    Write-Host "Available assets:"
    
    foreach ($ReleaseAsset in $ReleaseInfo.assets) {
        Write-Host "  $($ReleaseAsset.name)"
    }

    exit 1
}

$ZipName = $Asset.name
$DownloadUrl = $Asset.browser_download_url

$TempFolder = Join-Path $env:TEMP "seristack_install"

# Clean previous installation directory
if (Test-Path $TempFolder) {
    Remove-Item -Recurse -Force $TempFolder
}

New-Item -ItemType Directory -Force -Path $TempFolder | Out-Null

$ZipPath = Join-Path $TempFolder $ZipName

Write-Host "Downloading ${BinaryName} ${Tag}..."
Write-Host "Asset: $ZipName"

Invoke-WebRequest `
    -Uri $DownloadUrl `
    -OutFile $ZipPath

Write-Host "Extracting..."

Expand-Archive `
    -Path $ZipPath `
    -DestinationPath $TempFolder `
    -Force

# Install destination
$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\seristack"

New-Item `
    -ItemType Directory `
    -Force `
    -Path $InstallDir | Out-Null

$ExeSource = Join-Path $TempFolder "${BinaryName}.exe"
$ExeDestination = Join-Path $InstallDir "${BinaryName}.exe"

if (-not (Test-Path $ExeSource)) {
    Write-Error "Could not find $BinaryName.exe inside $ZipName"
    exit 1
}

Move-Item `
    -Path $ExeSource `
    -Destination $ExeDestination `
    -Force

# Add to User PATH
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")

if ($UserPath -notlike "*$InstallDir*") {

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

    $env:Path += ";$InstallDir"

    Write-Host "Added $InstallDir to User PATH."
}

Remove-Item `
    -Recurse `
    -Force `
    $TempFolder

Write-Host ""
Write-Host "Installation complete!"
Write-Host "Installed: $ExeDestination"
Write-Host ""

# Verify
& $ExeDestination --help

Write-Host ""
Write-Host "You may need to restart your terminal for PATH changes to take effect."
