$ErrorActionPreference = 'Stop'
$root = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
$fixture = Join-Path ([IO.Path]::GetTempPath()) ("qasmith tests " + [guid]::NewGuid().ToString('N'))
[IO.Directory]::CreateDirectory($fixture) | Out-Null
function Invoke-WebRequest {
    param([switch]$UseBasicParsing, [uri]$Uri, [string]$OutFile)
    if ($Uri.AbsoluteUri -notlike "https://github.com/quvly/Qasmith/releases/download/$tag/*") {
        throw "Unexpected asset URL $Uri"
    }
    Copy-Item -LiteralPath (Join-Path $fixture $Uri.Segments[-1]) -Destination $OutFile
}

try {
    $arch = if ([Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq 'Arm64') { 'arm64' } else { 'amd64' }
    $asset = "qasmith-windows-$arch.exe"
    $assetPath = Join-Path $fixture $asset
    $built = Join-Path $root 'qasmith.exe'
    $version = & $built version
    if ($LASTEXITCODE -ne 0 -or $version -notmatch '^Qasmith (v[0-9]+\.[0-9]+\.[0-9]+)$') { throw 'Unexpected build version' }
    $tag = $Matches[1]
    Copy-Item -LiteralPath $built -Destination $assetPath
    $hash = (Get-FileHash -LiteralPath $assetPath -Algorithm SHA256).Hash.ToLowerInvariant()
    Set-Content -LiteralPath "$assetPath.sha256" -Value "$hash  $asset"
    $installer = Join-Path $root 'scripts/install.ps1'
    $binDir = Join-Path $fixture 'bin with spaces'
    $installed = Join-Path $binDir 'qasmith.exe'
    & $installer -Version $tag -BinDir $binDir -NoPath
    & $installer -Version $tag -BinDir $binDir -NoPath
    if ((& $installed version) -ne $version) { throw 'Unexpected installed version' }
    Set-Content -LiteralPath $assetPath -Value 'bad download'
    try {
        & $installer -Version $tag -BinDir $binDir -NoPath
        throw 'Installer accepted a corrupt download'
    } catch {
        if ($_.Exception.Message -ne 'Qasmith download failed SHA-256 verification.') { throw }
    }
    if ((& $installed version) -ne $version) { throw 'Failed update changed installed executable' }
} finally {
    Remove-Item -LiteralPath $fixture -Force -Recurse
}
