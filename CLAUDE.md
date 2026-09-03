# CLAUDE.md

Guidance for agents working in this repo. Read `README.md` first for the user-facing story.

## What this is

A single-file Go daemon (`main.go`) that bridges an ATEN VS481C HDMI switch's RS232 port to HTTP and MQTT. It receives an input number (`1`-`4`) and writes `sw i0N\r\n` to the serial port at 19200 8N1. It runs on a Raspberry Pi under `systemd` in the author's setup.

## Layout

- `main.go` - the whole application: flag parsing, serial open, HTTP server, MQTT subscriber, command mapping
- `hdmi-switcher.service` - example systemd unit
- `README.md` - usage, configuration, Home Assistant integration
- `mise.toml` - pins the Go toolchain (`mise install`, then `mise exec -- go ...` or activate mise in your shell)

## Build and run

```
go build main.go
go run main.go --serial-path /dev/ttyUSB0 --mqtt-broker tcp://host:1883 --http-port 8080
go vet ./...
```

There are no tests. Running locally needs a real serial device and a reachable MQTT broker, or the process exits at startup.

### Cross-compiling for a Raspberry Pi

The target is a 32-bit Raspberry Pi OS (`uname -m` reports `armv7l`), so build with:

```
GOOS=linux GOARCH=arm GOARM=7 go build -o main .
```

Use `GOARCH=arm64` instead if the Pi is running a 64-bit OS (`aarch64`). The result is a static binary with no runtime dependencies; both Go modules are pure Go, so cgo is not needed. Copy it next to the existing binary, move it into place, then `sudo systemctl restart hdmi-switcher` and confirm the journal shows "Connected to serial port", "Connected to MQTT broker", and an "acknowledged" line after the next input change.

## Design decisions to preserve

- **Crash-and-restart resilience.** Loss of the MQTT connection deliberately calls `log.Fatalf`; systemd restarts the service. Do not add reconnect loops without discussion.
- **Every command waits for an acknowledgement.** The switch echoes the command followed by ` Command OK\r\n` within about 20ms; the app reads until the line terminator or a 500ms timeout and logs the result. No state is published back to MQTT.
- **Minimal dependencies.** Only `paho.mqtt.golang` and `go.bug.st/serial`. Requires Go 1.23 per `go.mod`; `http.ServeMux` path values (`req.PathValue`) need at least 1.22.
- **Any HTTP verb** is accepted on `/input/{id}`. Valid inputs reply `OK`, invalid ones reply `400`, and a missing or negative acknowledgement from the switch replies `502`.
- **Serial writes are serialized** through a mutex because the HTTP handler and MQTT callback run concurrently.

## Known rough edges

- Default flag values are specific to the author's network.
- Invalid MQTT payloads are logged and dropped; there is no feedback channel.

## Conventions

- Keep it a single file unless there's a strong reason otherwise.
- Prefer self-documenting code over comments.
- Update the README configuration table if you add or change a flag.
- Never commit on the author's behalf.
