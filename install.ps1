$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"   # the progress bar makes downloads very slow in Windows PowerShell 5.1

# GitHub requires TLS 1.2; older Windows PowerShell does not enable it by default.
try { [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 } catch { }

$Repo = "TechXploreLabs/seristack"
$BinaryName = "seristack"
$TempFolder = Join-Path ([IO.Path]::GetTempPath()) ("seristack_install_" + [guid]::NewGuid().ToString("N"))
$Failed = $false

try {

    # ------------------------------------------------------------
    # Detect Windows architecture (also correct inside 32-bit PowerShell)
    # ------------------------------------------------------------

    $RawArch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }

    $Arch = switch ($RawArch) {
        "AMD64" { "amd64" }
        "ARM64" { "arm64" }
        "x86"   { "386" }
        default { throw "Unsupported Windows architecture: $RawArch" }
    }

    Write-Host ""
    Write-Host "SeriStack Installer" -ForegroundColor Cyan
    Write-Host "--------------------"
    Write-Host "Architecture: $Arch"

    # ------------------------------------------------------------
    # Get latest GitHub release
    # ------------------------------------------------------------

    Write-Host ""
    Write-Host "Checking latest Seristack release..."

    $Headers = @{ "User-Agent" = "seristack-installer" }
    if ($env:GITHUB_TOKEN) { $Headers["Authorization"] = "Bearer $($env:GITHUB_TOKEN)" }

    try {
        $ReleaseInfo = Invoke-RestMethod -Headers $Headers `
            -Uri "https://api.github.com/repos/$Repo/releases/latest"
    }
    catch {
        $Status = $null
        if ($_.Exception.Response) { $Status = [int]$_.Exception.Response.StatusCode }
        if ($Status -eq 403 -or $Status -eq 429) {
            throw "GitHub API rate limit reached. Wait a while, or set the GITHUB_TOKEN environment variable and retry."
        }
        throw "Could not read the latest release from GitHub: $($_.Exception.Message)"
    }

    $Tag = $ReleaseInfo.tag_name

    if ([string]::IsNullOrWhiteSpace($Tag)) {
        throw "Unable to determine the latest release version."
    }

    $Version = $Tag -replace '^v', ''

    # ------------------------------------------------------------
    # Locate release assets
    # ------------------------------------------------------------

    $ArchiveName = "${BinaryName}_${Version}_windows_${Arch}.tar.gz"

    $Asset = $ReleaseInfo.assets |
        Where-Object { $_.name -eq $ArchiveName } |
        Select-Object -First 1

    if (-not $Asset) {

        Write-Host ""
        Write-Host "ERROR: Release asset not found." -ForegroundColor Red
        Write-Host "Expected: $ArchiveName"
        Write-Host ""
        Write-Host "Available assets:"

        foreach ($ReleaseAsset in $ReleaseInfo.assets) {
            Write-Host "  $($ReleaseAsset.name)"
        }

        throw "Windows release asset is unavailable."
    }

    $SumsAsset = $ReleaseInfo.assets |
        Where-Object { $_.name -match 'checksums\.txt$' } |
        Select-Object -First 1

    # ------------------------------------------------------------
    # Prepare temporary directory
    # ------------------------------------------------------------

    New-Item -ItemType Directory -Force -Path $TempFolder | Out-Null
    $ArchivePath = Join-Path $TempFolder $ArchiveName

    # ------------------------------------------------------------
    # Download
    # ------------------------------------------------------------

    Write-Host ""
    Write-Host "Downloading $BinaryName $Tag..." -ForegroundColor Cyan
    Write-Host "Asset: $ArchiveName"

    Invoke-WebRequest -Headers $Headers -Uri $Asset.browser_download_url -OutFile $ArchivePath

    if (-not (Test-Path $ArchivePath)) {
        throw "Download failed. Archive was not created."
    }

    # ------------------------------------------------------------
    # Verify checksum (when the release publishes a checksums file)
    # ------------------------------------------------------------

    if ($SumsAsset) {
        $SumsPath = Join-Path $TempFolder "checksums.txt"
        Invoke-WebRequest -Headers $Headers -Uri $SumsAsset.browser_download_url -OutFile $SumsPath

        $Pattern = "\s\*?" + [regex]::Escape($ArchiveName) + "$"
        $Line = Get-Content $SumsPath | Where-Object { $_ -match $Pattern } | Select-Object -First 1
        if (-not $Line) {
            throw "No checksum entry for $ArchiveName in $($SumsAsset.name)."
        }

        $Expected = (($Line.Trim()) -split '\s+')[0].ToLower()
        $Actual = (Get-FileHash -Path $ArchivePath -Algorithm SHA256).Hash.ToLower()

        if ($Expected -ne $Actual) {
            throw "Checksum mismatch for $ArchiveName (expected $Expected, got $Actual)."
        }
        Write-Host "Checksum verified." -ForegroundColor Green
    }
    else {
        Write-Host "WARNING: this release has no checksums file; the download could not be verified." -ForegroundColor Yellow
    }

    # ------------------------------------------------------------
    # Extract (use the Windows tar explicitly: Git's GNU tar breaks on C:\ paths)
    # ------------------------------------------------------------

    Write-Host ""
    Write-Host "Extracting archive..."

    $Tar = Join-Path $env:SystemRoot "System32\tar.exe"
    if (-not (Test-Path $Tar)) {
        throw "tar.exe was not found. It is included with Windows 10 (version 1803) and later."
    }

    & $Tar -xzf $ArchivePath -C $TempFolder

    if ($LASTEXITCODE -ne 0) {
        throw "Failed to extract the release archive."
    }

    # ------------------------------------------------------------
    # Locate executable (anywhere in the archive)
    # ------------------------------------------------------------

    $ExeItem = Get-ChildItem -Path $TempFolder -Recurse -Filter "${BinaryName}.exe" -File |
        Select-Object -First 1

    if (-not $ExeItem) {

        Write-Host ""
        Write-Host "Archive contents:" -ForegroundColor Yellow

        Get-ChildItem -Recurse $TempFolder | ForEach-Object { Write-Host "  $($_.FullName)" }

        throw "$BinaryName.exe was not found in the archive."
    }

    $ExeSource = $ExeItem.FullName

    # ------------------------------------------------------------
    # Install destination
    # ------------------------------------------------------------

    $InstallDir = Join-Path $env:LOCALAPPDATA "Programs\seristack"

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

    $ExeDestination = Join-Path $InstallDir "${BinaryName}.exe"

    # ------------------------------------------------------------
    # Install binary
    # ------------------------------------------------------------

    Write-Host ""
    Write-Host "Installing to:" -ForegroundColor Cyan
    Write-Host "  $InstallDir"

    try {
        Copy-Item -Path $ExeSource -Destination $ExeDestination -Force
    }
    catch [System.IO.IOException] {
        throw "Could not replace $ExeDestination because it is in use. Stop any running 'seristack' process (for example 'seristack run') and run the installer again."
    }

    # ------------------------------------------------------------
    # Add installation directory to User PATH
    # ------------------------------------------------------------

    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")

    $PathEntries = if ([string]::IsNullOrWhiteSpace($UserPath)) {
        @()
    }
    else {
        $UserPath -split ";" | ForEach-Object { $_.Trim().TrimEnd('\') }
    }

    $NormalizedInstallDir = $InstallDir.TrimEnd('\')

    $PathExists = $PathEntries | Where-Object { $_ -ieq $NormalizedInstallDir }

    if (-not $PathExists) {

        $NewUserPath = if ([string]::IsNullOrWhiteSpace($UserPath)) {
            $InstallDir
        }
        else {
            "$UserPath;$InstallDir"
        }

        [Environment]::SetEnvironmentVariable("Path", $NewUserPath, "User")

        Write-Host ""
        Write-Host "Added installation directory to User PATH." -ForegroundColor Green
    }
    else {
        Write-Host ""
        Write-Host "Installation directory already exists in User PATH."
    }

    # Update current PowerShell process PATH
    $CurrentPathEntries = $env:Path -split ";"

    if (-not ($CurrentPathEntries | Where-Object { $_.Trim().TrimEnd('\') -ieq $NormalizedInstallDir })) {
        $env:Path = "$env:Path;$InstallDir"
    }

    # ------------------------------------------------------------
    # Verify installation
    # ------------------------------------------------------------

    Write-Host ""
    Write-Host "Installation completed successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Version:    $Tag"
    Write-Host "Installed:  $ExeDestination"
    Write-Host ""

    Write-Host "Testing Seristack..." -ForegroundColor Cyan

    & $ExeDestination --help

    if ($LASTEXITCODE -ne 0) {
        Write-Host ""
        Write-Host "WARNING: Seristack was installed, but the verification command returned an error." -ForegroundColor Yellow
    }
    else {
        Write-Host ""
        Write-Host "Seristack is working!" -ForegroundColor Green
    }

    Write-Host ""
    Write-Host "You can now run:"
    Write-Host "  seristack"
    Write-Host ""
    Write-Host "If PATH is not recognized, restart your terminal."

}
catch {

    $Failed = $true

    Write-Host ""
    Write-Host "========================================" -ForegroundColor Red
    Write-Host " SERISTACK INSTALLATION FAILED" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red

    Write-Host ""
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
}
finally {

    # Always clean temporary files
    if (Test-Path $TempFolder) {
        Remove-Item -Recurse -Force $TempFolder -ErrorAction SilentlyContinue
    }

    # Report failure to callers without closing the user's terminal (no `exit`,
    # because this script may be run with `irm ... | iex`).
    if ($Failed) { $global:LASTEXITCODE = 1 }

    # Only pause when a person is at the console (not in CI or redirected input).
    if (-not [Console]::IsInputRedirected) {
        Write-Host ""
        Read-Host "Press Enter to finish"
    }
}
