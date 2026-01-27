Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Invoke-NativeCommand {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,
        [Parameter(Mandatory = $false)]
        [string[]]$Arguments = @()
    )

    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: $FilePath $($Arguments -join ' ')"
    }
}

function Convert-PngToIco {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PngPath,
        [Parameter(Mandatory = $true)]
        [string]$IcoPath
    )

    $pngBytes = [System.IO.File]::ReadAllBytes($PngPath)
    if ($pngBytes.Length -lt 24) {
        throw "PNG file is too small: $PngPath"
    }

    # Validate PNG signature.
    $expectedSig = [byte[]](137, 80, 78, 71, 13, 10, 26, 10)
    for ($i = 0; $i -lt $expectedSig.Length; $i++) {
        if ($pngBytes[$i] -ne $expectedSig[$i]) {
            throw "Invalid PNG signature: $PngPath"
        }
    }

    $width = [System.BitConverter]::ToUInt32([byte[]]($pngBytes[19], $pngBytes[18], $pngBytes[17], $pngBytes[16]), 0)
    $height = [System.BitConverter]::ToUInt32([byte[]]($pngBytes[23], $pngBytes[22], $pngBytes[21], $pngBytes[20]), 0)

    $iconWidth = if ($width -ge 256) { [byte]0 } else { [byte]$width }
    $iconHeight = if ($height -ge 256) { [byte]0 } else { [byte]$height }

    $stream = New-Object System.IO.MemoryStream
    $writer = New-Object System.IO.BinaryWriter($stream)
    try {
        # ICONDIR
        $writer.Write([UInt16]0)   # Reserved
        $writer.Write([UInt16]1)   # Type = icon
        $writer.Write([UInt16]1)   # One image

        # ICONDIRENTRY
        $writer.Write($iconWidth)
        $writer.Write($iconHeight)
        $writer.Write([byte]0)     # Palette size
        $writer.Write([byte]0)     # Reserved
        $writer.Write([UInt16]1)   # Color planes
        $writer.Write([UInt16]32)  # Bits per pixel
        $writer.Write([UInt32]$pngBytes.Length)
        $writer.Write([UInt32]22)  # Image data offset

        # PNG payload
        $writer.Write($pngBytes)

        [System.IO.File]::WriteAllBytes($IcoPath, $stream.ToArray())
    }
    finally {
        $writer.Dispose()
        $stream.Dispose()
    }
}

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$srcPath = Join-Path $root "cmd"
$iconPngPath = Join-Path $root "resources\logo\Logo_Optimal_Gradient.png"
$binPath = Join-Path $root "bin"
$exePath = Join-Path $binPath "EagleEye.exe"
$verifyScriptPath = Join-Path $root "verify_icon.ps1"
$goBinPath = if ([string]::IsNullOrWhiteSpace($env:GOBIN)) { Join-Path $env:USERPROFILE "go\bin" } else { $env:GOBIN }
$rsrcExe = Join-Path $goBinPath "rsrc.exe"
$tempIcoPath = Join-Path $env:TEMP "EagleEye_build_icon.ico"

$goExe = Join-Path $env:ProgramFiles "Go\bin\go.exe"
if (-not (Test-Path $goExe)) {
    throw "Go not found at $goExe. Install Go 1.21+ and retry."
}

if (-not (Test-Path $rsrcExe)) {
    Invoke-NativeCommand -FilePath $goExe -Arguments @("install", "github.com/akavel/rsrc@latest")
}

Invoke-NativeCommand -FilePath $goExe -Arguments @("mod", "tidy")
New-Item -ItemType Directory -Force $binPath | Out-Null

$resolvedIcon = (Resolve-Path $iconPngPath).Path
$goArch = (& $goExe env GOARCH).Trim()
if (-not $goArch) {
    throw "Unable to determine GOARCH."
}
$sysoPath = Join-Path $srcPath ("rsrc_windows_{0}.syso" -f $goArch)

Convert-PngToIco -PngPath $resolvedIcon -IcoPath $tempIcoPath
try {
    Invoke-NativeCommand -FilePath $rsrcExe -Arguments @("-ico", $tempIcoPath, "-arch", $goArch, "-o", $sysoPath)
    Invoke-NativeCommand -FilePath $goExe -Arguments @("build", "-ldflags", "-H=windowsgui", "-o", $exePath, "./cmd")
}
finally {
    if (Test-Path $tempIcoPath) {
        Remove-Item $tempIcoPath -Force
    }
    if (Test-Path $sysoPath) {
        Remove-Item $sysoPath -Force
    }
}

if (Test-Path $verifyScriptPath) {
    Invoke-NativeCommand -FilePath "powershell.exe" -Arguments @("-ExecutionPolicy", "Bypass", "-File", $verifyScriptPath, "-ExePath", $exePath)
}

Write-Host "Built: $exePath"
