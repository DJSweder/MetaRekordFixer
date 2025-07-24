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
go build -trimpath -buildvcs=false -ldflags "-w -s -H windowsgui -X 'MetaRekordFixer/common.dbPassword=402fd482c38817c35ffa8ffb8c7d93143b749e7d315df7a81732a1ff43608497'" -o "../dist/installer/sources/metarekordfixer.exe"


cd /d "%~dp0installer"
echo Creating installer...
iscc make_install_metarekordfixer-v1.0.0_inno.iss

echo Compilation and installer creation completed successfully.