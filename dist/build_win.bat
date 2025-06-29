@echo off

echo Setting the number of processor cores for compilation...
set GOMAXPROCS=8

echo Changing to src directory...
cd src

echo Updating dependencies...
go mod tidy

echo Changing back to root...
cd ..

echo Updating metadata and app icon...
go-winres make --in "dist/winres/winres.json"

echo Compiling the application...
set CGO_ENABLED=1
set CGO_CFLAGS=-w
go build -trimpath -buildvcs=false -ldflags "-w -s -H windowsgui" -o "dist/compilated/metarekordfixer.exe" src/main.go

echo Compressing the final binary file...
upx --best "dist/compilated/metarekordfixer.exe"

echo Compilation completed successfully.
