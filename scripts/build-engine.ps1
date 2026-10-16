param(
    [string]$GoExe = 'go'
)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$engine = Join-Path $root 'engine'
$bin = Join-Path $root 'bin'
New-Item -Path $bin -ItemType Directory -Force | Out-Null
Push-Location $engine
try {
    & $GoExe test ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go tests failed' }
    & $GoExe build -trimpath -o (Join-Path $bin 'bridge-core.exe') ./cmd/bridge-core
    if ($LASTEXITCODE -ne 0) { throw 'Go build failed' }
    Write-Host 'Native engine ready:' (Join-Path $bin 'bridge-core.exe')
}
finally { Pop-Location }
