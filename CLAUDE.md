# CLAUDE.md

Guidance for agents working in this repo. Read `README.md` first for the user-facing story.

## What this is

An ESPHome config for an ESP32 that bridges an ATEN VS481C HDMI switch's RS232 port to Home Assistant, with real state feedback and two indicator LEDs. It replaced a Go daemon on a Raspberry Pi, which lives on the `legacy` branch and is not developed further.

## Layout

- `hdmi-switch.yaml` - the ESPHome config; this is the product
- `esphome-build.md` - protocol notes, wiring gotchas, config outline, case constraints, sources
- `secrets.yaml` - Wi-Fi credentials for ESPHome, gitignored; never commit it or paste its contents
- `.github/workflows/build.yml` - CI: validates and compiles the config on PRs and pushes to main, with placeholder secrets

## Build and run

ESPHome, from the repo root so `secrets.yaml` is found:

```
esphome config hdmi-switch.yaml                                  # validate
esphome compile hdmi-switch.yaml                                 # build without flashing
esphome run hdmi-switch.yaml --device hdmi-switch.local          # OTA flash the live device
esphome logs hdmi-switch.yaml --device hdmi-switch.local         # stream logs
```

`esphome` is installed locally. Build output goes to `.esphome/`, which is gitignored. Always validate and compile before declaring a config change done; the device is in daily use, so only flash when the user asks.

There are no tests.

## Design decisions to preserve

- **The switch entities report truth, not intent.** They are non-optimistic and only change when the switch acknowledges a command or a `read` poll reports the input. Don't make them optimistic.
- **Commands and polls never overlap.** The `select_input` script records the time of every command and the 30 s `read` poll skips if one went out in the last 3 s. Keep that guard if you touch either.
- **One retry, then give up loudly.** An unacknowledged command is resent once after 600 ms with a log warning. Don't add more retries or silent ones.
- **The UART parser handles multi-line chunks.** The switch's `read` reply arrives in one chunk; the lambda splits on `\n` and trims `\r`. Don't assume one line per callback.
- **LED semantics are fixed:** green steady = HA connected, green blinking = no HA, red = switch silent for 90 s. Documented in the README; change both or neither.

## Conventions

- Keep the ESPHome config in one file.
- Prefer self-documenting config and code over comments.
- Update the README wiring or LED tables if pins or behaviour change.
- Never commit on the author's behalf.
- Never search or scan outside this repo for files; ask where things are.
