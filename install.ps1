$ErrorActionPreference = 'Stop'
$Repo = "cih1996/go-file-transfer"
$BinaryName = "jp-file.exe"
$DownloadUrl = "https://github.com/$Repo/releases/latest/download/jp-file-windows-amd64.exe"
$InstallDir = "$env:USERPROFILE\bin"

# Create installation directory if it doesn't exist
if (!(Test-Path -Path $InstallDir)) {
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
}

$OutputFile = Join-Path -Path $InstallDir -ChildPath $BinaryName

Write-Host "Downloading $BinaryName..."
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $OutputFile
} catch {
    Write-Error "Failed to download: $_"
    exit 1
}

Write-Host "Successfully installed to $OutputFile"

# Check if InstallDir is in PATH
$UserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($UserPath -notlike "*$InstallDir*") {
    Write-Host "Adding $InstallDir to User Path..."
    [Environment]::SetEnvironmentVariable('Path', $UserPath + ";$InstallDir", 'User')
    $env:PATH += ";$InstallDir"
    Write-Host "Path updated. You may need to restart your terminal."
}

Write-Host "Installation complete! Run 'jp-file' to start."
