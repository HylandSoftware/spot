# Spot - The Watchdog for your Build Agents

Spot is a watchdog for build agents in Jenkins and Bamboo

## Building

You need [`dep`](https://github.com/golang/dep). The easiest way to build
is to run `make`, which will generate linux and windows binaries in `dist/`.

If you don't have `make`, you can build manually:

```bash
# linux
dep ensure
go build -o dist/mmcmd -v ./cmd/mmcmd

# windows
dep ensure
go build -o dist/mmcmd.exe -v ./cmd/mmcmd
```