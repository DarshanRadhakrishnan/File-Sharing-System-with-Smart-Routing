# ============================================================
# P2P File Sharing System - Day 1 Integration Test
# Tests: Bootstrap server + 5 peer nodes + network graph
# ============================================================

$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  P2P File Sharing - Day 1 Test" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$projectRoot = $PSScriptRoot
$binDir = Join-Path $projectRoot "bin"
$bootstrapExe = Join-Path $binDir "bootstrap.exe"
$peerExe = Join-Path $binDir "peer.exe"

# ---- Build Phase ----
Write-Host "[1/5] Building binaries..." -ForegroundColor Yellow

if (-not (Test-Path $binDir)) {
    New-Item -ItemType Directory -Path $binDir -Force | Out-Null
}

Write-Host "  Building bootstrap server..."
& go build -o $bootstrapExe ./cmd/bootstrap
if ($LASTEXITCODE -ne 0) {
    Write-Host "  FAILED to build bootstrap!" -ForegroundColor Red
    exit 1
}

Write-Host "  Building peer application..."
& go build -o $peerExe ./cmd/peer
if ($LASTEXITCODE -ne 0) {
    Write-Host "  FAILED to build peer!" -ForegroundColor Red
    exit 1
}
Write-Host "  Build complete!" -ForegroundColor Green
Write-Host ""

# ---- Cleanup Function ----
$processes = @()

function Cleanup {
    Write-Host ""
    Write-Host "[5/5] Cleaning up processes..." -ForegroundColor Yellow
    foreach ($proc in $script:processes) {
        if (-not $proc.HasExited) {
            try {
                Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
                Write-Host "  Stopped PID $($proc.Id)"
            } catch {
                # Process already exited
            }
        }
    }
    Write-Host "  Cleanup complete!" -ForegroundColor Green
}

# Register cleanup on script exit
trap { Cleanup; break }

# ---- Start Bootstrap ----
Write-Host "[2/5] Starting bootstrap server on :5000..." -ForegroundColor Yellow
$bootstrap = Start-Process -FilePath $bootstrapExe -PassThru -NoNewWindow -RedirectStandardOutput (Join-Path $binDir "bootstrap_out.log") -RedirectStandardError (Join-Path $binDir "bootstrap_err.log")
$processes += $bootstrap
Start-Sleep -Seconds 2

if ($bootstrap.HasExited) {
    Write-Host "  Bootstrap server failed to start!" -ForegroundColor Red
    Get-Content (Join-Path $binDir "bootstrap_err.log")
    exit 1
}
Write-Host "  Bootstrap running (PID: $($bootstrap.Id))" -ForegroundColor Green
Write-Host ""

# ---- Start Peers ----
Write-Host "[3/5] Starting 5 peer nodes..." -ForegroundColor Yellow

$peerProcs = @()
for ($i = 1; $i -le 5; $i++) {
    $port = 9000 + $i
    $outLog = Join-Path $binDir "peer${i}_out.log"
    $errLog = Join-Path $binDir "peer${i}_err.log"
    
    Write-Host "  Starting peer-$i on port $port..."
    $proc = Start-Process -FilePath $peerExe -ArgumentList "--id=$i", "--port=$port", "--bootstrap=localhost:5000" -PassThru -NoNewWindow -RedirectStandardOutput $outLog -RedirectStandardError $errLog
    $processes += $proc
    $peerProcs += @{ Id = $i; Proc = $proc; OutLog = $outLog; ErrLog = $errLog }
    
    # Stagger peer startups to allow discovery
    Start-Sleep -Seconds 2
}

Write-Host "  All peers started!" -ForegroundColor Green
Write-Host ""

# ---- Wait for Discovery ----
Write-Host "[4/5] Waiting for peer discovery (5 seconds)..." -ForegroundColor Yellow
Start-Sleep -Seconds 5

# ---- Display Results ----
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  RESULTS" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Show bootstrap log
Write-Host "--- Bootstrap Server Log ---" -ForegroundColor Magenta
$bootstrapLog = Join-Path $binDir "bootstrap_out.log"
if (Test-Path $bootstrapLog) {
    Get-Content $bootstrapLog
}
Write-Host ""

# Show each peer's output
$allSuccess = $true
foreach ($peer in $peerProcs) {
    Write-Host "--- Peer-$($peer.Id) Output ---" -ForegroundColor Magenta
    if (Test-Path $peer.OutLog) {
        $content = Get-Content $peer.OutLog
        $content | ForEach-Object { Write-Host $_ }
        
        # Check for network graph in output
        if ($content -match "NETWORK GRAPH") {
            Write-Host "  [PASS] Network graph found!" -ForegroundColor Green
        } else {
            Write-Host "  [WARN] No network graph in output" -ForegroundColor Yellow
        }
    } else {
        Write-Host "  [FAIL] No output file found" -ForegroundColor Red
        $allSuccess = $false
    }
    
    # Check for errors
    if ((Test-Path $peer.ErrLog) -and (Get-Content $peer.ErrLog -ErrorAction SilentlyContinue)) {
        Write-Host "  Errors:" -ForegroundColor Red
        Get-Content $peer.ErrLog | ForEach-Object { Write-Host "    $_" -ForegroundColor Red }
        $allSuccess = $false
    }
    Write-Host ""
}

# ---- Summary ----
Write-Host "========================================" -ForegroundColor Cyan
if ($allSuccess) {
    Write-Host "  ALL TESTS PASSED!" -ForegroundColor Green
} else {
    Write-Host "  SOME TESTS FAILED!" -ForegroundColor Red
}
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Cleanup
Cleanup
