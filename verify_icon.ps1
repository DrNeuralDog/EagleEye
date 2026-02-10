param(
    [Parameter(Mandatory = $false)]
    [string]$ExePath = (Join-Path (Join-Path (Split-Path -Parent $MyInvocation.MyCommand.Path) "bin") "EagleEye.exe")
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if (-not (Test-Path $ExePath)) {
    Write-Error "Executable not found: $ExePath"
    exit 1
}

Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;

public static class Win32ResourceProbe
{
    private const uint LOAD_LIBRARY_AS_DATAFILE = 0x00000002;
    private static readonly IntPtr RT_ICON = (IntPtr)3;
    private static readonly IntPtr RT_GROUP_ICON = (IntPtr)14;

    [DllImport("kernel32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
    private static extern IntPtr LoadLibraryEx(string lpFileName, IntPtr hFile, uint dwFlags);

    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern bool FreeLibrary(IntPtr hModule);

    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern IntPtr FindResource(IntPtr hModule, IntPtr lpName, IntPtr lpType);

    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern IntPtr FindResource(IntPtr hModule, string lpName, IntPtr lpType);

    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern bool EnumResourceNames(
        IntPtr hModule,
        IntPtr lpszType,
        EnumResNameProc lpEnumFunc,
        IntPtr lParam
    );

    private delegate bool EnumResNameProc(IntPtr hModule, IntPtr lpszType, IntPtr lpszName, IntPtr lParam);

    private static int CountResourcesByType(IntPtr module, IntPtr type)
    {
        int count = 0;
        EnumResNameProc callback = (h, t, name, l) =>
        {
            count++;
            return true;
        };
        EnumResourceNames(module, type, callback, IntPtr.Zero);
        return count;
    }

    public static int CountGroupIcons(string path)
    {
        IntPtr module = LoadLibraryEx(path, IntPtr.Zero, LOAD_LIBRARY_AS_DATAFILE);
        if (module == IntPtr.Zero)
        {
            throw new InvalidOperationException("LoadLibraryEx failed.");
        }

        try
        {
            return CountResourcesByType(module, RT_GROUP_ICON);
        }
        finally
        {
            FreeLibrary(module);
        }
    }

    public static int CountIcons(string path)
    {
        IntPtr module = LoadLibraryEx(path, IntPtr.Zero, LOAD_LIBRARY_AS_DATAFILE);
        if (module == IntPtr.Zero)
        {
            throw new InvalidOperationException("LoadLibraryEx failed.");
        }

        try
        {
            return CountResourcesByType(module, RT_ICON);
        }
        finally
        {
            FreeLibrary(module);
        }
    }
}
"@

$resolvedExe = (Resolve-Path $ExePath).Path
$groupIconCount = [Win32ResourceProbe]::CountGroupIcons($resolvedExe)
$iconCount = [Win32ResourceProbe]::CountIcons($resolvedExe)

if ($groupIconCount -le 0 -or $iconCount -le 0) {
    Write-Error "Icon resource is missing in executable: $resolvedExe (RT_GROUP_ICON=$groupIconCount, RT_ICON=$iconCount)"
    exit 1
}

Write-Host "Icon resource check passed: $resolvedExe (RT_GROUP_ICON=$groupIconCount, RT_ICON=$iconCount)"
