#!/bin/bash

mkdir -p out/win64
# Call sudo early so that user can type in their password sooner rather than later.
sudo ls 2>&1 > /dev/null
go run build/grab-win64-resources-on-wsl.go
env CGO_ENABLED="1" CC="/usr/bin/x86_64-w64-mingw32-gcc" GOOS="windows" GOARCH="amd64" CGO_LDFLAGS="-lmingw32 -lSDL2" CGO_FLAGS="-D_REENTRANT" go build -o build.exe
mv build.exe out/win64
cp -r assets out/win64
# This assumes that the main partition is installed on the C drive, and that the usernames are the same (I think WSL ensures this?)
# WSL does not assume this. Allow the user to specify in param 1
cp -r out/win64 ${1:-/mnt/c/Users/$(whoami)/Desktop}
