param
(
    [Parameter(Mandatory = $false)]
    [switch]$RunGoModTidy,
    [Parameter(Mandatory = $false)]
    [switch]$AllowGoNetwork,
    [Parameter(Mandatory = $false)]
    [int]$CommandTimeoutSeconds = 600
)


Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"


function Get-EffectiveCommandTimeout {
    param
    (
        [Parameter(Mandatory = $true)]
        [int]$TimeoutSeconds
    )

    if ($TimeoutSeconds -lt 30) { return 30 }


    return $TimeoutSeconds
}


function Format-CommandArguments {
    param
    (
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
    param
    (
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
    param
    (
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
    param
    (
        [Parameter(Mandatory = $true)]
        [string]$PngPath,
        [Parameter(Mandatory = $true)]
        [string]$IcoPath
    )


    $pngBytes = [System.IO.File]::ReadAllBytes($PngPath)

    if ($pngBytes.Length -lt 24) {
        throw "PNG file is too small: $PngPath"
    }


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
        $writer.Write([UInt16]0)
        $writer.Write([UInt16]1)
        $writer.Write([UInt16]1)

        $writer.Write($iconWidth)
        $writer.Write($iconHeight)
        $writer.Write([byte]0)
        $writer.Write([byte]0)
        $writer.Write([UInt16]1)
        $writer.Write([UInt16]32)
        $writer.Write([UInt32]$pngBytes.Length)
        $writer.Write([UInt32]22)

        $writer.Write($pngBytes)

        [System.IO.File]::WriteAllBytes($IcoPath, $stream.ToArray())
    }


    finally {
        $writer.Dispose()
        $stream.Dispose()
    }
}


function New-BuildContext {
    param
    (
        [Parameter(Mandatory = $true)]
        [string]$ScriptPath
    )

    
    $root = Split-Path -Parent $ScriptPath
    $binPath = Join-Path $root "bin"
    $goBinPath = if ([string]::IsNullOrWhiteSpace($env:GOBIN)) { Join-Path $env:USERPROFILE "go\bin" } else { $env:GOBIN }

    return [PSCustomObject]@{
        Root             = $root
        SourcePath       = Join-Path $root "cmd"
        IconPngPath      = Join-Path $root "resources\logo\Logo_Optimal_Gradient.png"
        BinPath          = $binPath
        ExePath          = Join-Path $binPath "EagleEye.exe"
        VerifyScriptPath = Join-Path $root "verify_icon.ps1"
        RsrcExe          = Join-Path $goBinPath "rsrc.exe"
        TempIcoPath      = Join-Path $env:TEMP "EagleEye_build_icon.ico"
    }
}


function Set-GoModuleNetwork {
    param
    (
        [Parameter(Mandatory = $true)]
        [switch]$AllowGoNetwork
    )


    $goProxyMode = if ($AllowGoNetwork) { "https://proxy.golang.org,direct" } else { "off" }
    $goSumDBMode = if ($AllowGoNetwork) { "sum.golang.org" } else { "off" }
    $env:GOPROXY = $goProxyMode
    $env:GOSUMDB = $goSumDBMode
    
    Write-Host "Go module network: GOPROXY=$goProxyMode; GOSUMDB=$goSumDBMode"
}


function Initialize-GoToolchain {
    param
    (
        [Parameter(Mandatory = $true)]
        [switch]$AllowGoNetwork,
        [Parameter(Mandatory = $true)]
        [int]$CommandTimeoutSeconds
    )


    $goExe = Resolve-GoExecutable
    Assert-GoMinimumVersion -GoExe $goExe -MinimumMinor 21

    Write-Host "Using Go executable: $goExe"
    Write-Host "Command timeout: $CommandTimeoutSeconds seconds"

    Set-GoModuleNetwork -AllowGoNetwork:$AllowGoNetwork


    return $goExe
}


function Ensure-RsrcExecutable {
    param(
        [Parameter(Mandatory = $true)]
        [string]$RsrcExe,
        [Parameter(Mandatory = $true)]
        [string]$GoExe,
        [Parameter(Mandatory = $true)]
        [switch]$AllowGoNetwork
    )


    if (Test-Path $RsrcExe) {
        return
    }


    if (-not $AllowGoNetwork) {
        throw "rsrc.exe not found at '$RsrcExe', and Go network is disabled. Re-run once with -AllowGoNetwork."
    }


    Write-Host "rsrc.exe not found, installing..."
    
    Invoke-NativeCommand -FilePath $GoExe -Arguments @("install", "github.com/akavel/rsrc@latest")
}


function Invoke-GoModTidyStep {
    param
    (
        [Parameter(Mandatory = $true)]
        [string]$GoExe,
        [Parameter(Mandatory = $true)]
        [switch]$RunGoModTidy
    )


    if ($RunGoModTidy) {
        Write-Host "Running go mod tidy..."
        Invoke-NativeCommand -FilePath $GoExe -Arguments @("mod", "tidy")

        return
    }


    Write-Host "Skipping go mod tidy (enable with -RunGoModTidy)."
}


function Ensure-OutputDirectory {
    param
    (
        [Parameter(Mandatory = $true)]
        [string]$BinPath
    )


    New-Item -Path $BinPath -ItemType Directory -Force | Out-Null
}

function Resolve-GoArchitecture {
    param
    (
        [Parameter(Mandatory = $true)]
        [string]$GoExe
    )


    $goArch = (& $GoExe env GOARCH).Trim()
    
    if (-not $goArch) {
        throw "Unable to determine GOARCH."
    }


    return $goArch
}


function Remove-TemporaryBuildArtifacts {
    param
    (
        [Parameter(Mandatory = $true)]
        [string[]]$Paths
    )


    foreach ($path in $Paths) {
        if (Test-Path $path) {
            Remove-Item $path -Force
        }
    }
}


function Invoke-WindowsExecutableBuild {
    param
    (
        [Parameter(Mandatory = $true)]
        [PSCustomObject]$Context,
        [Parameter(Mandatory = $true)]
        [string]$GoExe,
        [Parameter(Mandatory = $true)]
        [string]$GoArch
    )


    $resolvedIcon = (Resolve-Path $Context.IconPngPath).Path
    $sysoPath = Join-Path $Context.SourcePath ("rsrc_windows_{0}.syso" -f $GoArch)

    Convert-PngToIco -PngPath $resolvedIcon -IcoPath $Context.TempIcoPath
   
    try {
        Invoke-NativeCommand -FilePath $Context.RsrcExe -Arguments @("-ico", $Context.TempIcoPath, "-arch", $GoArch, "-o", $sysoPath)
        Invoke-NativeCommand -FilePath $GoExe -Arguments @("build", "-buildvcs=false", "-ldflags", "-H=windowsgui", "-o", $Context.ExePath, "./cmd")
    }
    finally {
        Remove-TemporaryBuildArtifacts -Paths @($Context.TempIcoPath, $sysoPath)
    }
}


function Invoke-IconVerification {
    param
    (
        [Parameter(Mandatory = $true)]
        [PSCustomObject]$Context
    )


    if (Test-Path $Context.VerifyScriptPath) {
        Invoke-NativeCommand -FilePath "powershell.exe" -Arguments @("-ExecutionPolicy", "Bypass", "-File", $Context.VerifyScriptPath, "-ExePath", $Context.ExePath)
    }
}


function Invoke-EagleEyeBuild {
    param
    (
        [Parameter(Mandatory = $true)]
        [string]$ScriptPath,
        [Parameter(Mandatory = $true)]
        [switch]$RunGoModTidy,
        [Parameter(Mandatory = $true)]
        [switch]$AllowGoNetwork,
        [Parameter(Mandatory = $true)]
        [int]$CommandTimeoutSeconds
    )


    $script:CommandTimeoutSeconds = Get-EffectiveCommandTimeout -TimeoutSeconds $CommandTimeoutSeconds
    $context = New-BuildContext -ScriptPath $ScriptPath
    $goExe = Initialize-GoToolchain -AllowGoNetwork:$AllowGoNetwork -CommandTimeoutSeconds $script:CommandTimeoutSeconds

    Ensure-RsrcExecutable -RsrcExe $context.RsrcExe -GoExe $goExe -AllowGoNetwork:$AllowGoNetwork
    Invoke-GoModTidyStep -GoExe $goExe -RunGoModTidy:$RunGoModTidy
    Ensure-OutputDirectory -BinPath $context.BinPath

    $goArch = Resolve-GoArchitecture -GoExe $goExe
    Invoke-WindowsExecutableBuild -Context $context -GoExe $goExe -GoArch $goArch
    Invoke-IconVerification -Context $context

    Write-Host "Built: $($context.ExePath)"
}


Invoke-EagleEyeBuild -ScriptPath $MyInvocation.MyCommand.Path -RunGoModTidy:$RunGoModTidy -AllowGoNetwork:$AllowGoNetwork -CommandTimeoutSeconds $CommandTimeoutSeconds
