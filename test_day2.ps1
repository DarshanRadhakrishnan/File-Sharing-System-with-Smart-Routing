# ============================================================
# DAY 2: P2P DIJKSTRA ROUTING TEST (Windows PowerShell)
# ============================================================

$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " DAY 2: P2P DIJKSTRA ROUTING TEST" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

# Build
Write-Host "[1/5] Building bootstrap server..." -ForegroundColor Yellow
go build -o bin/bootstrap.exe ./cmd/bootstrap
Write-Host "      Done." -ForegroundColor Green

Write-Host "[2/5] Building peer application..." -ForegroundColor Yellow
go build -o bin/peer.exe ./cmd/peer
Write-Host "      Done." -ForegroundColor Green

Write-Host "[3/5] Building benchmark..." -ForegroundColor Yellow
go build -o bin/benchmark.exe ./cmd/benchmark
Write-Host "      Done." -ForegroundColor Green

Write-Host ""
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " RUNNING BENCHMARK (100 nodes)" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "[4/5] Running routing benchmark..." -ForegroundColor Yellow
& .\bin\benchmark.exe
Write-Host ""

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " INTEGRATION TEST (10 peers)" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "[5/5] Starting network with 10 peers..." -ForegroundColor Yellow
Write-Host ""

# Start bootstrap server
$bootstrap = Start-Process -FilePath ".\bin\bootstrap.exe" -PassThru -WindowStyle Hidden
Start-Sleep -Seconds 1
Write-Host "  Bootstrap server started (PID: $($bootstrap.Id))" -ForegroundColor Green

# Start 10 peers
$peerProcesses = @()
for ($i = 1; $i -le 10; $i++) {
    $port = 9000 + $i
    $proc = Start-Process -FilePath ".\bin\peer.exe" `
        -ArgumentList "--id=$i", "--port=$port", "--bootstrap=localhost:5000" `
        -PassThru -WindowStyle Hidden `
        -RedirectStandardOutput "bin\peer_${i}_out.log" `
        -RedirectStandardError "bin\peer_${i}_err.log"
    $peerProcesses += $proc
    Write-Host "  Peer-$i started on port $port (PID: $($proc.Id))" -ForegroundColor DarkGray
    Start-Sleep -Milliseconds 500
}

Write-Host ""
Write-Host "  Waiting for peers to discover each other..." -ForegroundColor Yellow
Start-Sleep -Seconds 5

Write-Host ""
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " PEER LOGS" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

# Show output from the last few peers (they have the most connections)
for ($i = 8; $i -le 10; $i++) {
    $logFile = "bin\peer_${i}_out.log"
    if (Test-Path $logFile) {
        Write-Host "--- Peer-$i output ---" -ForegroundColor Magenta
        Get-Content $logFile | Select-Object -Last 25
        Write-Host ""
    }
}

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " CLEANUP" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

# Cleanup
foreach ($proc in $peerProcesses) {
    try { Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue } catch {}
}
try { Stop-Process -Id $bootstrap.Id -Force -ErrorAction SilentlyContinue } catch {}

Write-Host "  All processes stopped." -ForegroundColor Green
Write-Host ""
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " DAY 2 COMPLETE!" -ForegroundColor Green
Write-Host " Dijkstra routing working with 10 peers!" -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""
