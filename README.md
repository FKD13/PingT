# Ping'T

*A TUI application graphically visualizing the ping status of a target*

![](./screenshot.png)

---

This application acts as a wrapper for `fping`.
This means all target formats accepted by your installed `fping` binary should also be accepted by this tool.

Currently, the program accepts the following flags:

| Flag               | Description                                                |
|--------------------|------------------------------------------------------------|
| `-t` or `--target` | A target to ping. Should be repeated once for each target. |
| `--refresh`        | Time is ms between redraws of the grid.                    |



## Installation

First install the `fping` program per instruction of your operating system. 
For systems using apt `apt install fping` should be sufficient.

Then either build the program yourself as instructed bellow or download it from the [releases](https://github.com/FKD13/PingTUI/releases) page.
Place the binary in a location that is included in your PATH.

Recreate the above example by running the following command
```bash
pingt -t 1.1.1.1 -t 1.0.0.1 -t 8.8.8.8 -t 6.9.6.9 -t ::1 -t github.com -t gitlab.com
```

## Building

```bash
go build -o pingt -ldflags="-s -w" .
```