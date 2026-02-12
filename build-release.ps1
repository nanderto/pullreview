#!/usr/bin/env pwsh
# Build script for creating release binaries

param(
    [string]$Version = "0.1.0"
)

$ErrorActionPreference = "Stop"

Write-Host "Building pullreview v$Version for release..." -ForegroundColor Cyan

# Clean previous builds
if (Test-Path "releases") {
    Write-Host "Cleaning previous releases..." -ForegroundColor Yellow
    Remove-Item -Recurse -Force releases
}

New-Item -ItemType Directory -Path releases | Out-Null

# Build configuration
$builds = @(
    @{OS="windows"; ARCH="amd64"; EXT=".exe"}
    @{OS="windows"; ARCH="arm64"; EXT=".exe"}
    @{OS="linux"; ARCH="amd64"; EXT=""}
    @{OS="linux"; ARCH="arm64"; EXT=""}
    @{OS="darwin"; ARCH="amd64"; EXT=""}
    @{OS="darwin"; ARCH="arm64"; EXT=""}
)

# Build flags for optimization
$ldflags = "-s -w -X main.version=$Version"

foreach ($build in $builds) {
    $os = $build.OS
    $arch = $build.ARCH
    $ext = $build.EXT
    
    # Create platform-specific directory
    $platformDir = "releases/$os-$arch"
    New-Item -ItemType Directory -Path $platformDir -Force | Out-Null
    $output = "$platformDir/pullreview$ext"
    
    Write-Host "`nBuilding $os/$arch..." -ForegroundColor Green
    
    $env:GOOS = $os
    $env:GOARCH = $arch
    $env:CGO_ENABLED = "0"
    
    go build -ldflags $ldflags -trimpath -o $output cmd/pullreview/main.go
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Build failed for $os/$arch" -ForegroundColor Red
        exit 1
    }
    
    $size = (Get-Item $output).Length
    $sizeKB = [math]::Round($size / 1KB, 2)
    $sizeMB = [math]::Round($size / 1MB, 2)
    
    if ($sizeMB -ge 1) {
        Write-Host "  ✓ Created: $output ($sizeMB MB)" -ForegroundColor Green
    } else {
        Write-Host "  ✓ Created: $output ($sizeKB KB)" -ForegroundColor Green
    }
}

# Reset environment variables
Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue

Write-Host "`n✅ Release build complete!" -ForegroundColor Cyan
Write-Host "Binaries available in: releases/" -ForegroundColor Cyan

# List all builds
Write-Host "`nBuilt binaries:" -ForegroundColor Yellow
Get-ChildItem releases | ForEach-Object {
    $size = [math]::Round($_.Length / 1MB, 2)
    Write-Host "  - $($_.Name) ($size MB)"
}
