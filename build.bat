@echo off
echo ==========================================
echo  正在构建 Windows 定时关机小工具...
echo ==========================================

REM 1. 编译 Windows 资源与清单 (启用 DPI 感知与现代界面样式)
windres -i app.rc -o rsrc.syso -O coff
if %errorlevel% neq 0 (
    echo [错误] windres 资源编译失败！
    pause
    exit /b %errorlevel%
)

REM 2. 使用 Go 编译无控制台黑框的 Windows GUI 可执行程序
go build -ldflags="-H windowsgui -s -w" -o win-auto-shutdown.exe .
if %errorlevel% neq 0 (
    echo [错误] Go 构建失败！
    pause
    exit /b %errorlevel%
)

echo.
echo ==========================================
echo  构建成功！生成文件: win-auto-shutdown.exe
echo ==========================================
pause
