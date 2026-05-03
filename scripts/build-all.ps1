param(
    [string]$Version = "1.0.0",
    [string]$OutDir = "dist"
)

$ErrorActionPreference = "Stop"

$archs = @("amd64", "386", "arm64")
$ldflags = "-s -w -X main.Version=$Version"

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

foreach ($arch in $archs) {
    $out = "$OutDir\windows-portable-packager-$arch.exe"
    $env:GOARCH = $arch
    $env:GOOS = "windows"
    Write-Host "Building $arch -> $out"
    go build -ldflags "$ldflags" -o $out .
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Build failed for $arch"
        exit 1
    }
    $size = [math]::Round((Get-Item $out).Length / 1MB, 2)
    Write-Host "  OK ($size MB)"
}

Remove-Item Env:GOARCH
Remove-Item Env:GOOS

Write-Host "`nAll builds complete:"
foreach ($arch in $archs) {
    $out = "$OutDir\windows-portable-packager-$arch.exe"
    $size = [math]::Round((Get-Item $out).Length / 1MB, 2)
    Write-Host "  $out ($size MB)"
}
