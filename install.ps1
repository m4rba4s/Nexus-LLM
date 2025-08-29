# GOLLM Installation Script for Windows PowerShell
# Usage: irm https://raw.githubusercontent.com/yourusername/gollm/main/install.ps1 | iex

param(
    [string]$InstallDir = "",
    [string]$Version = "",
    [switch]$Force,
    [switch]$Help
)

# Script configuration
$Repo = "yourusername/gollm"
$BinaryName = "gollm.exe"
$DefaultInstallDir = "$env:LOCALAPPDATA\Programs\GOLLM"

# Color configuration for output
$Colors = @{
    Info    = "Cyan"
    Success = "Green"
    Warning = "Yellow"
    Error   = "Red"
    Header  = "Magenta"
}

# Function to print colored output
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Type = "Info"
    )

    $color = $Colors[$Type]
    $prefix = switch ($Type) {
        "Info"    { "ℹ" }
        "Success" { "✓" }
        "Warning" { "⚠" }
        "Error"   { "✗" }
        default   { "•" }
    }

    Write-Host "$prefix $Message" -ForegroundColor $color
}

function Write-Header {
    param([string]$Title)

    $line = "━" * 75
    Write-Host ""
    Write-Host $line -ForegroundColor Magenta
    Write-Host ("{0,-75}" -f "                        $Title") -ForegroundColor Magenta
    Write-Host $line -ForegroundColor Magenta
    Write-Host ""
}

function Show-Help {
    Write-Host "GOLLM Installation Script for Windows" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "USAGE:" -ForegroundColor Yellow
    Write-Host "    .\install.ps1 [options]"
    Write-Host ""
    Write-Host "OPTIONS:" -ForegroundColor Yellow
    Write-Host "    -InstallDir <path>    Installation directory (default: $DefaultInstallDir)"
    Write-Host "    -Version <version>    Specific version to install (default: latest)"
    Write-Host "    -Force               Force installation even if already installed"
    Write-Host "    -Help                Show this help message"
    Write-Host ""
    Write-Host "EXAMPLES:" -ForegroundColor Yellow
    Write-Host "    .\install.ps1"
    Write-Host "    .\install.ps1 -InstallDir 'C:\Tools\GOLLM'"
    Write-Host "    .\install.ps1 -Version 'v1.0.0'"
    Write-Host ""
}

function Test-AdminPrivileges {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default {
            Write-ColorOutput "Unsupported architecture: $arch" "Error"
            Write-ColorOutput "Supported: amd64, arm64" "Info"
            exit 1
        }
    }
}

function Get-LatestVersion {
    try {
        Write-ColorOutput "Getting latest version from GitHub..."
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Method Get
        return $response.tag_name
    }
    catch {
        Write-ColorOutput "Failed to get latest version: $($_.Exception.Message)" "Error"
        Write-ColorOutput "Please check your internet connection and try again" "Info"
        exit 1
    }
}

function Download-File {
    param(
        [string]$Url,
        [string]$OutputPath
    )

    try {
        Write-ColorOutput "Downloading from: $Url"

        # Use BITS transfer if available (faster and more reliable)
        if (Get-Command Start-BitsTransfer -ErrorAction SilentlyContinue) {
            Start-BitsTransfer -Source $Url -Destination $OutputPath -DisplayName "GOLLM Download"
        }
        else {
            # Fallback to Invoke-WebRequest
            $ProgressPreference = 'SilentlyContinue'
            Invoke-WebRequest -Uri $Url -OutFile $OutputPath -UseBasicParsing
            $ProgressPreference = 'Continue'
        }

        return $true
    }
    catch {
        Write-ColorOutput "Failed to download file: $($_.Exception.Message)" "Error"
        return $false
    }
}

function Test-Checksum {
    param(
        [string]$FilePath,
        [string]$ChecksumUrl
    )

    try {
        Write-ColorOutput "Verifying checksum..."

        # Download checksum file
        $checksumFile = "$env:TEMP\gollm_checksum.txt"
        if (-not (Download-File -Url $ChecksumUrl -OutputPath $checksumFile)) {
            Write-ColorOutput "Could not download checksum file, skipping verification" "Warning"
            return $true
        }

        # Read expected checksum
        $expectedChecksum = (Get-Content $checksumFile -Raw).Split()[0].Trim()

        # Calculate actual checksum
        $actualChecksum = (Get-FileHash -Path $FilePath -Algorithm SHA256).Hash.ToLower()

        # Clean up checksum file
        Remove-Item $checksumFile -Force -ErrorAction SilentlyContinue

        if ($expectedChecksum.ToLower() -eq $actualChecksum) {
            Write-ColorOutput "Checksum verification passed" "Success"
            return $true
        }
        else {
            Write-ColorOutput "Checksum verification failed" "Error"
            Write-ColorOutput "Expected: $expectedChecksum" "Error"
            Write-ColorOutput "Actual:   $actualChecksum" "Error"
            return $false
        }
    }
    catch {
        Write-ColorOutput "Checksum verification failed: $($_.Exception.Message)" "Warning"
        return $true # Don't fail installation on checksum issues
    }
}

function Expand-Archive {
    param(
        [string]$ArchivePath,
        [string]$DestinationPath
    )

    try {
        Write-ColorOutput "Extracting archive..."

        # Ensure destination directory exists
        if (-not (Test-Path $DestinationPath)) {
            New-Item -ItemType Directory -Path $DestinationPath -Force | Out-Null
        }

        # Extract ZIP archive
        Expand-Archive -Path $ArchivePath -DestinationPath $DestinationPath -Force

        Write-ColorOutput "Archive extracted successfully" "Success"
        return $true
    }
    catch {
        Write-ColorOutput "Failed to extract archive: $($_.Exception.Message)" "Error"
        return $false
    }
}

function Install-Binary {
    param(
        [string]$BinaryPath,
        [string]$InstallPath
    )

    try {
        Write-ColorOutput "Installing binary to $InstallPath"

        # Create install directory if it doesn't exist
        $installDir = Split-Path $InstallPath -Parent
        if (-not (Test-Path $installDir)) {
            New-Item -ItemType Directory -Path $installDir -Force | Out-Null
        }

        # Copy binary
        Copy-Item -Path $BinaryPath -Destination $InstallPath -Force

        Write-ColorOutput "Binary installed successfully" "Success"
        return $true
    }
    catch {
        Write-ColorOutput "Failed to install binary: $($_.Exception.Message)" "Error"
        return $false
    }
}

function Add-ToPath {
    param([string]$Directory)

    try {
        # Get current user PATH
        $currentPath = [Environment]::GetEnvironmentVariable("PATH", [EnvironmentVariableTarget]::User)

        # Check if directory is already in PATH
        if ($currentPath -split ';' | Where-Object { $_.TrimEnd('\') -eq $Directory.TrimEnd('\') }) {
            Write-ColorOutput "Directory already in PATH: $Directory" "Info"
            return $true
        }

        # Add to PATH
        $newPath = if ($currentPath) { "$currentPath;$Directory" } else { $Directory }
        [Environment]::SetEnvironmentVariable("PATH", $newPath, [EnvironmentVariableTarget]::User)

        # Update current session PATH
        $env:PATH = "$env:PATH;$Directory"

        Write-ColorOutput "Added to PATH: $Directory" "Success"
        return $true
    }
    catch {
        Write-ColorOutput "Failed to add directory to PATH: $($_.Exception.Message)" "Error"
        return $false
    }
}

function Install-Completion {
    param([string]$BinaryPath)

    Write-ColorOutput "Setting up PowerShell completion..."

    try {
        # Check if PowerShell profile exists
        if (-not (Test-Path $PROFILE)) {
            Write-ColorOutput "Creating PowerShell profile: $PROFILE" "Info"
            New-Item -Path $PROFILE -Type File -Force | Out-Null
        }

        # Check if completion is already added
        $profileContent = Get-Content $PROFILE -Raw -ErrorAction SilentlyContinue
        if ($profileContent -and $profileContent -match "gollm completion powershell") {
            Write-ColorOutput "PowerShell completion already configured" "Info"
            return $true
        }

        # Add completion to profile
        $completionLine = "& '$BinaryPath' completion powershell | Out-String | Invoke-Expression"
        Add-Content -Path $PROFILE -Value "`n# GOLLM completion`n$completionLine" -Encoding UTF8

        Write-ColorOutput "PowerShell completion installed" "Success"
        Write-ColorOutput "Restart PowerShell or run: . `$PROFILE" "Info"
        return $true
    }
    catch {
        Write-ColorOutput "Failed to install completion: $($_.Exception.Message)" "Warning"
        return $false
    }
}

function Test-Installation {
    param([string]$BinaryPath)

    Write-ColorOutput "Testing installation..."

    try {
        # Test if binary exists and is executable
        if (-not (Test-Path $BinaryPath)) {
            Write-ColorOutput "Binary not found at: $BinaryPath" "Error"
            return $false
        }

        # Test version command
        $versionOutput = & $BinaryPath version --short 2>$null
        if ($LASTEXITCODE -eq 0 -and $versionOutput) {
            Write-ColorOutput "Installation test passed" "Success"
            Write-ColorOutput "GOLLM version: $versionOutput" "Info"
            return $true
        }
        else {
            Write-ColorOutput "Installation test failed" "Error"
            return $false
        }
    }
    catch {
        Write-ColorOutput "Installation test failed: $($_.Exception.Message)" "Error"
        return $false
    }
}

function Show-UsageInfo {
    Write-Host ""
    $line = "━" * 75
    Write-Host $line -ForegroundColor Cyan
    Write-Host ("{0,-75}" -f "                              Getting Started") -ForegroundColor Cyan
    Write-Host $line -ForegroundColor Cyan
    Write-Host ""

    Write-Host "1. Initialize configuration:" -ForegroundColor Green
    Write-Host "   gollm config init" -ForegroundColor White
    Write-Host ""

    Write-Host "2. Set up your API keys:" -ForegroundColor Green
    Write-Host "   gollm config set providers.openai.api_key `"sk-your-api-key`"" -ForegroundColor White
    Write-Host "   gollm config set providers.anthropic.api_key `"your-anthropic-key`"" -ForegroundColor White
    Write-Host ""

    Write-Host "3. Test your first completion:" -ForegroundColor Green
    Write-Host "   gollm chat `"Hello, world!`"" -ForegroundColor White
    Write-Host ""

    Write-Host "4. Start interactive mode:" -ForegroundColor Green
    Write-Host "   gollm interactive" -ForegroundColor White
    Write-Host ""

    Write-Host "5. Get help:" -ForegroundColor Green
    Write-Host "   gollm --help" -ForegroundColor White
    Write-Host "   gollm [command] --help" -ForegroundColor White
    Write-Host ""

    Write-Host "Documentation: " -NoNewline -ForegroundColor Cyan
    Write-Host "https://docs.gollm.dev" -ForegroundColor White

    Write-Host "GitHub:        " -NoNewline -ForegroundColor Cyan
    Write-Host "https://github.com/$Repo" -ForegroundColor White

    Write-Host "Issues:        " -NoNewline -ForegroundColor Cyan
    Write-Host "https://github.com/$Repo/issues" -ForegroundColor White
    Write-Host ""
}

function Main {
    # Show help if requested
    if ($Help) {
        Show-Help
        return
    }

    Write-Header "GOLLM Installation Script"

    # Set installation directory
    if (-not $InstallDir) {
        $InstallDir = $DefaultInstallDir
    }

    # Check if already installed
    $binaryPath = Join-Path $InstallDir $BinaryName
    if ((Test-Path $binaryPath) -and -not $Force) {
        Write-ColorOutput "GOLLM is already installed at: $binaryPath" "Warning"
        Write-ColorOutput "Use -Force to reinstall or -Help for options" "Info"
        return
    }

    # Detect architecture
    Write-ColorOutput "Detecting system architecture..."
    $arch = Get-Architecture
    Write-ColorOutput "Architecture detected: $arch" "Success"

    # Get version to install
    if (-not $Version) {
        $Version = Get-LatestVersion
    }
    Write-ColorOutput "Version to install: $Version" "Success"

    # Prepare download URLs
    $versionNoV = $Version.TrimStart('v')
    $archiveName = "gollm-$versionNoV-windows-$arch.zip"
    $downloadUrl = "https://github.com/$Repo/releases/download/$Version/$archiveName"
    $checksumUrl = "$downloadUrl.sha256"

    # Create temporary directory
    $tempDir = Join-Path $env:TEMP "gollm-install-$(Get-Random)"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

    try {
        $archivePath = Join-Path $tempDir $archiveName

        # Download archive
        if (-not (Download-File -Url $downloadUrl -OutputPath $archivePath)) {
            Write-ColorOutput "Failed to download GOLLM" "Error"
            return
        }
        Write-ColorOutput "Download completed" "Success"

        # Verify checksum
        Test-Checksum -FilePath $archivePath -ChecksumUrl $checksumUrl

        # Extract archive
        $extractDir = Join-Path $tempDir "extract"
        if (-not (Expand-Archive -ArchivePath $archivePath -DestinationPath $extractDir)) {
            return
        }

        # Find binary in extracted files
        $tempBinaryPath = Join-Path $extractDir $BinaryName
        if (-not (Test-Path $tempBinaryPath)) {
            Write-ColorOutput "Binary not found in archive" "Error"
            return
        }

        # Install binary
        if (-not (Install-Binary -BinaryPath $tempBinaryPath -InstallPath $binaryPath)) {
            return
        }

        # Add to PATH
        $installDir = Split-Path $binaryPath -Parent
        Add-ToPath -Directory $installDir

        # Install PowerShell completion
        Install-Completion -BinaryPath $binaryPath

        # Test installation
        if (-not (Test-Installation -BinaryPath $binaryPath)) {
            Write-ColorOutput "Installation completed but tests failed" "Warning"
            Write-ColorOutput "You may need to restart PowerShell or check your PATH" "Info"
        }
        else {
            Write-ColorOutput "GOLLM installed successfully!" "Success"
        }

        # Show usage information
        Show-UsageInfo
    }
    finally {
        # Cleanup temporary directory
        if (Test-Path $tempDir) {
            Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

# Check PowerShell version
if ($PSVersionTable.PSVersion.Major -lt 5) {
    Write-Host "PowerShell 5.0 or later is required" -ForegroundColor Red
    Write-Host "Current version: $($PSVersionTable.PSVersion)" -ForegroundColor Red
    exit 1
}

# Check execution policy
$executionPolicy = Get-ExecutionPolicy
if ($executionPolicy -eq "Restricted") {
    Write-Host "PowerShell execution policy is set to Restricted" -ForegroundColor Red
    Write-Host "Run: Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser" -ForegroundColor Yellow
    exit 1
}

# Run main function with error handling
try {
    Main
}
catch {
    Write-ColorOutput "Installation failed: $($_.Exception.Message)" "Error"
    Write-ColorOutput "Stack trace: $($_.ScriptStackTrace)" "Error"
    exit 1
}
