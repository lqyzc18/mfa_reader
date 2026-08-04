@echo off
chcp 65001 >nul
echo ==========================================
echo   虚拟 MFA 桌面端工具 - Windows 打包脚本
echo ==========================================
echo.

set "FYNE_CMD="
where fyne >nul 2>&1
if %errorlevel% equ 0 (
    set "FYNE_CMD=fyne"
) else if exist "%USERPROFILE%\go\bin\fyne.exe" (
    set "FYNE_CMD=%USERPROFILE%\go\bin\fyne.exe"
) else if exist "%GOPATH%\bin\fyne.exe" (
    set "FYNE_CMD=%GOPATH%\bin\fyne.exe"
)

if "%FYNE_CMD%"=="" (
    echo [错误] fyne.exe 未找到！
    echo 请先安装 Fyne 工具链：
    echo   go install fyne.io/fyne/v2/cmd/fyne@latest
    echo.
    pause
    exit /b 1
)

echo [步骤 1/2] 清理旧构建文件...
if exist "mfa_reader.exe" (
    del "mfa_reader.exe"
    echo   - 已删除旧的 mfa_reader.exe
)

echo [步骤 2/2] 正在打包 Windows 版本...
echo.
"%FYNE_CMD%" package -os windows -icon icon.png

if %errorlevel% neq 0 (
    echo.
    echo [错误] 打包失败！错误代码: %errorlevel%
    pause
    exit /b 1
)

echo.
echo ==========================================
echo   打包成功！
echo ==========================================
echo.
echo 生成的文件: mfa_reader.exe
echo.
echo 提示:
echo   - 双击运行 mfa_reader.exe 即可启动应用
echo   - 如需带图标的安装包，请使用专业打包工具
echo.
pause
