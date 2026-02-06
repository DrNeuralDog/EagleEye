Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$srcPath = Join-Path $root "cmd"
$iconPath = Join-Path $root "resources\logo\Logo_Optimal_Gradient.png"
$binPath = Join-Path $root "bin"
$exePath = Join-Path $binPath "EagleEye.exe"

$goExe = Join-Path $env:ProgramFiles "Go\bin\go.exe"
if (-not (Test-Path $goExe)) {
    throw "Go not found at $goExe. Install Go 1.21+ and retry."
}

$fyneExe = Join-Path $env:USERPROFILE "go\bin\fyne.exe"
if (-not (Test-Path $fyneExe)) {
    & $goExe install fyne.io/tools/cmd/fyne@latest
}

& $goExe mod tidy
New-Item -ItemType Directory -Force $binPath | Out-Null
& $goExe build -ldflags "-H=windowsgui" -o $exePath ./cmd

$resolvedSrc = (Resolve-Path $srcPath).Path
$resolvedIcon = (Resolve-Path $iconPath).Path

& $fyneExe package --os windows --exe $exePath --src $resolvedSrc --icon $resolvedIcon --name "EagleEye" --app-id "com.eagleeye.app" --release

Write-Host "Built: $exePath"
