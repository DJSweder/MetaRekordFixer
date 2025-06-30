@echo off

echo Setting the number of processor cores for compilation...
set GOMAXPROCS=8

echo Cleaning old resource files...
del /q "%~dp0..\src\*_windows_*.syso"

echo Changing to src directory...
cd /d "%~dp0..\src"

echo Updating dependencies...
go mod tidy

echo Updating metadata and app icon...
cd /d "%~dp0.."
go-winres make --in "dist/winres/winres.json" --out "src/rsrc"

cd /d "%~dp0..\src"
echo Compiling the application...
set CGO_ENABLED=1
set CGO_CFLAGS=-w
go build -trimpath -buildvcs=false -ldflags "-w -s -H windowsgui" -o "../dist/compilated/metarekordfixer.exe" main.go

REM echo Compressing the final binary file...
REM cd /d "%~dp0.."
REM upx --best "dist/compilated/metarekordfixer.exe"

echo Compilation completed successfully.