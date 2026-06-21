# MyNote 开发模式启动脚本
# 同时启动后端 Go 服务和前端 Vite 开发服务器

$ErrorActionPreference = "Stop"
$rootDir = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  MyNote - 开发模式启动" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 设置环境变量
$env:MYNOTE_DATA_DIR = Join-Path $rootDir "backend" "data"
$env:GIN_MODE = "debug"

# 创建数据目录
$dataDir = Join-Path $rootDir "backend" "data"
if (-not (Test-Path $dataDir)) {
    New-Item -ItemType Directory -Path $dataDir -Force | Out-Null
    Write-Host "[OK] 创建数据目录: $dataDir" -ForegroundColor Green
}

# 启动后端（后台运行）
Write-Host "[启动] 后端服务 (Go)" -ForegroundColor Yellow
$backendJob = Start-Job -ScriptBlock {
    param($dir)
    Set-Location $dir
    go run main.go
} -ArgumentList (Join-Path $rootDir "backend")

Write-Host "[OK] 后端服务已启动 (PID: $($backendJob.Id))" -ForegroundColor Green

# 启动前端
Write-Host "[启动] 前端开发服务器 (Vite)" -ForegroundColor Yellow
Set-Location (Join-Path $rootDir "frontend")
$frontendProcess = Start-Process -FilePath "npx.cmd" -ArgumentList "vite" -NoNewWindow -PassThru -WorkingDirectory (Join-Path $rootDir "frontend")

Write-Host "[OK] 前端开发服务器已启动 (PID: $($frontendProcess.Id))" -ForegroundColor Green
Write-Host ""
Write-Host "------------------------------------------------" -ForegroundColor Cyan
Write-Host "  后端 API:  http://localhost:8080" -ForegroundColor Green
Write-Host "  前端页面:  http://localhost:3000" -ForegroundColor Green
Write-Host "  数据目录:  $dataDir" -ForegroundColor Green
Write-Host "------------------------------------------------" -ForegroundColor Cyan
Write-Host ""
Write-Host "按 Ctrl+C 停止所有服务..." -ForegroundColor Gray
Write-Host ""

# 等待前端进程结束
try {
    $frontendProcess.WaitForExit()
} finally {
    # 清理
    Write-Host "[停止] 正在停止服务..." -ForegroundColor Yellow
    Stop-Job $backendJob -ErrorAction SilentlyContinue
    Remove-Job $backendJob -ErrorAction SilentlyContinue
    if (-not $frontendProcess.HasExited) {
        $frontendProcess.Kill()
    }
    Write-Host "[OK] 所有服务已停止" -ForegroundColor Green
}
