$ErrorActionPreference = 'Stop'
$Repo = "cih1996/go-file-transfer"

if ($env:TARGET) {
    $Target = $env:TARGET
} else {
    $Target = "jp-file"
}

$BinaryName = "$Target.exe"
$GithubUrl = "https://github.com/$Repo/releases/latest/download/${Target}-windows-amd64.exe"
$MirrorUrl = "https://github-1308564197.cos.ap-guangzhou.myqcloud.com/go-file-transfer/latest/${Target}-windows-amd64.exe"
$InstallDir = "$env:USERPROFILE\bin"

# Create installation directory if it doesn't exist
if (!(Test-Path -Path $InstallDir)) {
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
}

$OutputFile = Join-Path -Path $InstallDir -ChildPath $BinaryName

Function Download-File {
    param ([string]$Url, [string]$OutFile)
    try {
        Invoke-WebRequest -Uri $Url -OutFile $OutFile -ErrorAction Stop
        return $true
    } catch {
        return $false
    }
}

Write-Host "Downloading $BinaryName..."
Write-Host "Trying domestic mirror ($MirrorUrl)..."

if (-not (Download-File -Url $MirrorUrl -OutFile $OutputFile)) {
    Write-Host "Mirror failed. Trying official GitHub ($GithubUrl)..."
    if (-not (Download-File -Url $GithubUrl -OutFile $OutputFile)) {
        Write-Error "Failed to download from both sources."
        exit 1
    }
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

Write-Host "Installation complete! Run '$Target' to start."
