[CmdletBinding()]
param(
    [ValidatePattern('^(latest|v[0-9]+\.[0-9]+\.[0-9]+)$')]
    [string]$Version = 'latest',
    [string]$BinDir = (Join-Path $env:LOCALAPPDATA 'Qasmith\bin'),
    [switch]$NoPath
)

$ErrorActionPreference = 'Stop'
$architecture = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()) {
    'X64' { 'amd64' }
    'Arm64' { 'arm64' }
    default { throw 'Qasmith supports x86-64 and ARM64 Windows.' }
}
$base = 'https://github.com/quvly/Qasmith/releases'
$url = if ($Version -eq 'latest') { "$base/latest/download" } else { "$base/download/$Version" }
$asset = "qasmith-windows-$architecture.exe"
$BinDir = [System.IO.Path]::GetFullPath($BinDir)
[System.IO.Directory]::CreateDirectory($BinDir) | Out-Null
$stage = Join-Path $BinDir ('.qasmith-' + [guid]::NewGuid().ToString('N'))
[System.IO.Directory]::CreateDirectory($stage) | Out-Null
try {
    $binary = Join-Path $stage $asset
    $checksum = Join-Path $stage "$asset.sha256"
    Invoke-WebRequest -UseBasicParsing -Uri "$url/$asset.sha256" -OutFile $checksum
    Invoke-WebRequest -UseBasicParsing -Uri "$url/$asset" -OutFile $binary
    $expected = (Get-Content -LiteralPath $checksum -Raw).Trim() -split '\s+'
    $actual = (Get-FileHash -LiteralPath $binary -Algorithm SHA256).Hash
    if ($expected.Count -ne 2 -or $expected[1] -cne $asset -or $expected[0] -ne $actual) {
        throw 'Qasmith download failed SHA-256 verification.'
    }
    & $binary version
    if ($LASTEXITCODE -ne 0) { throw "Qasmith version check exited with $LASTEXITCODE." }
    $destination = Join-Path $BinDir 'qasmith.exe'
    if ([System.IO.File]::Exists($destination)) {
        [System.IO.File]::Replace($binary, $destination, [NullString]::Value)
    } else {
        [System.IO.File]::Move($binary, $destination)
    }
    if (-not $NoPath) {
        $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
        if (($userPath -split ';') -notcontains $BinDir) {
            $newPath = ($userPath, $BinDir | Where-Object { $_ }) -join ';'
            [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        }
        if (($env:Path -split ';') -notcontains $BinDir) {
            $env:Path = "$env:Path;$BinDir"
        }
    }
    Write-Host "Installed $destination"
} finally {
    Remove-Item -LiteralPath $stage -Recurse -Force
}
