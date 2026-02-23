param(
    [Parameter(Mandatory = $false)]
    [switch]$RunGoModTidy,
    [Parameter(Mandatory = $false)]
    [switch]$AllowGoNetwork,
    [Parameter(Mandatory = $false)]
    [int]$CommandTimeoutSeconds = 600
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
if ($CommandTimeoutSeconds -lt 30) {
    $CommandTimeoutSeconds = 30
}
$script:CommandTimeoutSeconds = $CommandTimeoutSeconds

function Format-CommandArguments {
    param(
        [Parameter(Mandatory = $false)]
        [string[]]$Arguments = @()
    )

    return ($Arguments | ForEach-Object {
            if ($_ -match '[\s"]') {
                '"' + ($_ -replace '"', '\"') + '"'
            }
            else {
                $_
            }
        }) -join ' '
}

function Invoke-NativeCommand {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,
        [Parameter(Mandatory = $false)]
        [string[]]$Arguments = @(),
        [Parameter(Mandatory = $false)]
        [int]$TimeoutSeconds = $script:CommandTimeoutSeconds
    )

    $argumentLine = Format-CommandArguments -Arguments $Arguments
    Write-Host ">> $FilePath $argumentLine"

    $startInfo = New-Object System.Diagnostics.ProcessStartInfo
    $startInfo.FileName = $FilePath
    $startInfo.Arguments = $argumentLine
    $startInfo.UseShellExecute = $false
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    $startInfo.CreateNoWindow = $true

    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $startInfo

    if (-not $process.Start()) {
        throw "Failed to start command: $FilePath $argumentLine"
    }

    $stdoutTask = $process.StandardOutput.ReadToEndAsync()
    $stderrTask = $process.StandardError.ReadToEndAsync()

    if (-not $process.WaitForExit($TimeoutSeconds * 1000)) {
        try {
            $process.Kill()
        }
        catch {
            # no-op
        }
        throw "Command timed out after ${TimeoutSeconds}s: $FilePath $argumentLine"
    }
    $process.WaitForExit()

    $stdout = $stdoutTask.GetAwaiter().GetResult()
    if (-not [string]::IsNullOrWhiteSpace($stdout)) {
        Write-Host ($stdout.TrimEnd())
    }

    $stderr = $stderrTask.GetAwaiter().GetResult()
    if (-not [string]::IsNullOrWhiteSpace($stderr)) {
        Write-Host ($stderr.TrimEnd())
    }

    if ($process.ExitCode -ne 0) {
        throw "Command failed with exit code $($process.ExitCode): $FilePath $argumentLine"
    }
}

function Resolve-GoExecutable {
    $candidates = @()

    if (-not [string]::IsNullOrWhiteSpace($env:GOROOT)) {
        $candidates += (Join-Path $env:GOROOT "bin\go.exe")
    }

    try {
        $goCommand = Get-Command "go" -ErrorAction Stop
        if ($goCommand -and -not [string]::IsNullOrWhiteSpace($goCommand.Source)) {
            $candidates += $goCommand.Source
        }
    }
    catch {
        # no-op: fallback candidates below
    }

    if (-not [string]::IsNullOrWhiteSpace($env:ProgramFiles)) {
        $candidates += (Join-Path $env:ProgramFiles "Go\bin\go.exe")
    }

    $uniqueCandidates = $candidates | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Select-Object -Unique
    foreach ($candidate in $uniqueCandidates) {
        if (Test-Path $candidate) {
            return $candidate
        }
    }

    throw "Go executable not found. Install Go 1.21+ or add 'go.exe' to PATH."
}

function Assert-GoMinimumVersion {
    param(
        [Parameter(Mandatory = $true)]
        [string]$GoExe,
        [Parameter(Mandatory = $true)]
        [int]$MinimumMinor
    )

    $goVersionStr = (& $GoExe env GOVERSION).Trim()
    if ([string]::IsNullOrWhiteSpace($goVersionStr)) {
        throw "Unable to read Go version from '$GoExe' (go env GOVERSION)."
    }

    if ($goVersionStr -match '^go(\d+)\.(\d+)') {
        $major = [int]$Matches[1]
        $minor = [int]$Matches[2]
        if ($major -gt 1) {
            return
        }
        if ($major -eq 1 -and $minor -ge $MinimumMinor) {
            return
        }
        throw @"
Go 1.$MinimumMinor+ required (go.mod). Found $goVersionStr from: $GoExe

Install a current toolchain from https://go.dev/dl/
If GOROOT points at an old install, update or remove GOROOT after installing.
"@
    }

    throw "Unrecognized go version string: $goVersionStr"
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

$goExe = Resolve-GoExecutable
Assert-GoMinimumVersion -GoExe $goExe -MinimumMinor 21
Write-Host "Using Go executable: $goExe"
Write-Host "Command timeout: $CommandTimeoutSeconds seconds"
$goProxyMode = if ($AllowGoNetwork) { "https://proxy.golang.org,direct" } else { "off" }
$goSumDBMode = if ($AllowGoNetwork) { "sum.golang.org" } else { "off" }
$env:GOPROXY = $goProxyMode
$env:GOSUMDB = $goSumDBMode
Write-Host "Go module network: GOPROXY=$goProxyMode; GOSUMDB=$goSumDBMode"

if (-not (Test-Path $rsrcExe)) {
    if (-not $AllowGoNetwork) {
        throw "rsrc.exe not found at '$rsrcExe', and Go network is disabled. Re-run once with -AllowGoNetwork."
    }
    Write-Host "rsrc.exe not found, installing..."
    Invoke-NativeCommand -FilePath $goExe -Arguments @("install", "github.com/akavel/rsrc@latest")
}

if ($RunGoModTidy) {
    Write-Host "Running go mod tidy..."
    Invoke-NativeCommand -FilePath $goExe -Arguments @("mod", "tidy")
}
else {
    Write-Host "Skipping go mod tidy (enable with -RunGoModTidy)."
}
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
    Invoke-NativeCommand -FilePath $goExe -Arguments @("build", "-buildvcs=false", "-ldflags", "-H=windowsgui", "-o", $exePath, "./cmd")
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
