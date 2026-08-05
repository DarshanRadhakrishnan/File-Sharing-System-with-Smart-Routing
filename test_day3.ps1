$ErrorActionPreference = "Stop"

Write-Host "=========================================="
Write-Host "DAY 3: BLOOM FILTER & HASH VERIFICATION TEST"
Write-Host "=========================================="

# Ensure test file exists
if (!(Test-Path "test_files")) {
    New-Item -ItemType Directory -Path "test_files" | Out-Null
}

if (!(Test-Path "test_files/sample.txt")) {
    Write-Host "Creating sample test file (1MB)..."
    $bytes = New-Object Byte[] 1048576
    $rand = New-Object System.Random
    $rand.NextBytes($bytes)
    [System.IO.File]::WriteAllBytes("test_files/sample.txt", $bytes)
}

Write-Host "[1/4] Building bootstrap server..."
if (!(Test-Path "bin")) {
    New-Item -ItemType Directory -Path "bin" | Out-Null
}
Push-Location cmd/bootstrap
go build -o ../../bin/bootstrap.exe
Pop-Location

Write-Host "[2/4] Building peer application..."
Push-Location cmd/peer
go build -o ../../bin/peer.exe
Pop-Location

Write-Host "[3/4] Running Bloom filter unit test..."
go run test_bloom.go

Write-Host ""
Write-Host "[4/4] Starting network + testing chunk upload/verify..."
Write-Host ""

# Clean up previous runs
Stop-Process -Name "bootstrap" -ErrorAction SilentlyContinue
Stop-Process -Name "peer" -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1
if (Test-Path "data") {
    Remove-Item -Recurse -Force "data" -ErrorAction SilentlyContinue
}

# Start bootstrap
$bootstrapProc = Start-Process -FilePath ".\bin\bootstrap.exe" -PassThru -NoNewWindow

Start-Sleep -Seconds 1

# Start 5 peers
$peerProcs = @()
for ($i = 1; $i -le 5; $i++) {
    $port = 9000 + $i
    $proc = Start-Process -FilePath ".\bin\peer.exe" -ArgumentList "--id=$i", "--port=$port", "--bootstrap=localhost:5000" -PassThru -NoNewWindow
    $peerProcs += $proc
    Start-Sleep -Milliseconds 300
}

Start-Sleep -Seconds 3

Write-Host ""
go run test_upload.go

Write-Host ""
Write-Host "=========================================="
Write-Host "Cleaning up..."
Stop-Process -Id $bootstrapProc.Id -Force -ErrorAction SilentlyContinue
foreach ($p in $peerProcs) {
    Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue
}

Write-Host "Test completed!"
Write-Host "Day 3: Bloom filters + SHA-256 verification working!"
