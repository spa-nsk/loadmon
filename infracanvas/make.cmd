@echo off
setlocal
cd /d %~dp0

set OUT=%~dp0..\server_web_new\static\infracanvas
if not exist "%OUT%" mkdir "%OUT%"

echo [infracanvas] go mod tidy
go mod tidy
if errorlevel 1 exit /b 1

echo [infracanvas] build WASM
set GOOS=js
set GOARCH=wasm
go build -o "%OUT%\infracanvas.wasm" .
if errorlevel 1 exit /b 1

for /f "delims=" %%G in ('go env GOROOT') do set GOROOT=%%G
if exist "%GOROOT%\misc\wasm\wasm_exec.js" (
  copy /Y "%GOROOT%\misc\wasm\wasm_exec.js" "%OUT%\wasm_exec.js" >nul
) else if exist "%GOROOT%\lib\wasm\wasm_exec.js" (
  copy /Y "%GOROOT%\lib\wasm\wasm_exec.js" "%OUT%\wasm_exec.js" >nul
) else (
  echo WARNING: wasm_exec.js not found in GOROOT
)

echo [infracanvas] done: %OUT%\infracanvas.wasm
endlocal
