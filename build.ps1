#!/usr/bin/env pwsh
# Simple build script for local development

param(
    [string]$Version = "0.1.0-dev"
)

$ErrorActionPreference = "Stop"

Write-Host "Building pullreview v$Version..." -ForegroundColor Cyan

# Clean previous build
if (Test-Path "pullreview.exe") {
    Remove-Item pullreview.exe
}

# Build flags
$ldflags = "-X main.version=$Version"

go build -ldflags $ldflags -o pullreview.exe cmd/pullreview/main.go

if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed" -ForegroundColor Red
    exit 1
}

$size = (Get-Item pullreview.exe).Length
$sizeMB = [math]::Round($size / 1MB, 2)

Write-Host "✅ Build complete: pullreview.exe ($sizeMB MB)" -ForegroundColor Green
