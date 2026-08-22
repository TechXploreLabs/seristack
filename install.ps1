$ErrorActionPreference = "Stop"

$Repo = "TechXploreLabs/seristack"
$BinaryName = "seristack"

# Detect Architecture
$Arch = if ([Environment]::Is64BitProcess) { "amd64" } else { "386" }

# Get latest release version
$ReleaseInfo = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
$Tag = $ReleaseInfo.tag_name
$Version = $Tag.TrimStart("v")

$ZipName = "${BinaryName}_${Version}_windows_${Arch}.zip"
$DownloadUrl = "https://github.com/$Repo/releases/download/${Tag}/${ZipName}"

$TempFolder = Join-Path $env:TEMP "seristack_install"
New-Item -ItemType Directory -Force -Path $TempFolder | Out-Null

$ZipPath = Join-Path $TempFolder $ZipName
Write-Host "Downloading ${BinaryName} ${Tag}..."
Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath

Write-Host "Extracting..."
Expand-Archive -Path $ZipPath -DestinationPath $TempFolder -Force

# Install destination (Local AppData to avoid requiring Admin rights)
$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\seristack"
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

Move-Item -Path (Join-Path $TempFolder "${BinaryName}.exe") -Destination (Join-Path $InstallDir "${BinaryName}.exe") -Force

# Add to User PATH if not already present
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path += ";$InstallDir"
    Write-Host "Added $InstallDir to User PATH."
}

Remove-Item -Recurse -Force $TempFolder

Write-Host "Installation complete! You may need to restart your terminal."
seristack --help