@echo off

echo Setting the number of processor cores for compilation...
set GOMAXPROCS=8

cd /d "%~dp0..\src"

echo Cleaning old resource files...
del /q *.syso

echo Generating icon and metadata...
go-winres make --in "%~dp0..\dist\winres\winres.json" --out "rsrc"

echo Updating dependencies...
go mod tidy

echo Compiling the application...
set CGO_ENABLED=1
set CGO_CFLAGS=-w
go build -trimpath -buildvcs=false -ldflags "-w -s -H windowsgui" -o "..\dist\compilated\metarekordfixer.exe"

REM echo Compressing the final binary file...
REM cd /d "%~dp0.."
REM upx --best "dist/compilated/metarekordfixer.exe"

echo Compilation completed successfully.