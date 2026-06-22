# MyNote 构建脚本
# 构建前端和后端，生成生产环境部署包

$ErrorActionPreference = "Stop"
$rootDir = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$outputDir = Join-Path $rootDir "build"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  MyNote - 生产构建" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 1. 构建前端
Write-Host "[1/3] 构建前端..." -ForegroundColor Yellow
Set-Location (Join-Path $rootDir "frontend")
npm run build
if ($LASTEXITCODE -ne 0) {
    Write-Host "[失败] 前端构建失败" -ForegroundColor Red
    exit 1
}
Write-Host "[OK] 前端构建完成" -ForegroundColor Green

# 2. 构建后端
Write-Host "[2/3] 构建后端..." -ForegroundColor Yellow
Set-Location (Join-Path $rootDir "backend")
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -ldflags="-s -w" -o mynote-server.exe .
if ($LASTEXITCODE -ne 0) {
    Write-Host "[失败] 后端构建失败" -ForegroundColor Red
    exit 1
}
Write-Host "[OK] 后端构建完成" -ForegroundColor Green

# 3. 打包
Write-Host "[3/3] 打包部署文件..." -ForegroundColor Yellow
if (Test-Path $outputDir) {
    Remove-Item -Recurse -Force $outputDir
}
New-Item -ItemType Directory -Path $outputDir -Force | Out-Null

# 复制后端
Copy-Item (Join-Path $rootDir "backend\mynote-server.exe") $outputDir

# 复制配置文件
Copy-Item (Join-Path $rootDir "backend\config.yaml") $outputDir

# 复制前端编译产物
Copy-Item -Recurse (Join-Path $rootDir "frontend\dist") (Join-Path $outputDir "dist")

# 创建数据目录
New-Item -ItemType Directory -Path (Join-Path $outputDir "data") -Force | Out-Null

# 创建启动脚本
@"
@echo off
set MYNOTE_DATA_DIR=%~dp0data
set MYNOTE_DIST_DIR=%~dp0dist
start /B mynote-server.exe
echo MyNote 已启动，访问 http://localhost:8080
pause
"@ | Out-File -FilePath (Join-Path $outputDir "start.bat") -Encoding ASCII

Write-Host "[OK] 打包完成" -ForegroundColor Green
Write-Host ""
Write-Host "------------------------------------------------" -ForegroundColor Cyan
Write-Host "  部署目录: $outputDir" -ForegroundColor Green
Write-Host "  运行方式: 双击 start.bat 或执行 mynote-server.exe" -ForegroundColor Green
Write-Host "  访问地址: http://localhost:8080" -ForegroundColor Green
Write-Host "------------------------------------------------" -ForegroundColor Cyan
