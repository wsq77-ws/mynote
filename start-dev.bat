@echo off
chcp 65001 >nul
echo ========================================
echo   MyNote - 开发模式启动
echo ========================================
echo.

:: 设置环境变量
set "MYNOTE_DATA_DIR=%~dp0backend\data"

:: 确保数据目录存在
if not exist "%~dp0backend\data" mkdir "%~dp0backend\data"

:: 启动后端（新窗口）
echo [启动] 后端服务 (Go) ...
start "MyNote-Backend" cmd /c "cd /d %~dp0backend && go run main.go"

:: 等待后端启动
timeout /t 3 /nobreak >nul

:: 启动前端
echo [启动] 前端开发服务器 (Vite) ...
start "MyNote-Frontend" cmd /c "cd /d %~dp0frontend && npx vite"

echo.
echo ========================================
echo   后端 API:  http://localhost:8080
echo   前端页面:  http://localhost:3000
echo ========================================
echo.
echo 按任意键关闭所有服务...
pause >nul

:: 关闭启动的窗口
taskkill /fi "WindowTitle eq MyNote-Backend*" /f >nul 2>&1
taskkill /fi "WindowTitle eq MyNote-Frontend*" /f >nul 2>&1
