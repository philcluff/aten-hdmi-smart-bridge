# ESPHome build notes

Reference for the ESP32 build described in the [README](README.md): what was learned about the switch, the wiring gotchas, the ESPHome config, and the case constraints. Built and working since 2026-09-05.

## What we know about the switch

Confirmed from a USB bus trace of the original Pi setup and the VS481C manual:

| Setting | Value |
|---|---|
| Serial | 19200 baud, 8 data bits, no parity, 1 stop bit, no flow control |
| Command terminator | `\r\n` |
| Switch input | `sw i01` to `sw i04` |
| Acknowledgement | The command echoed back followed by ` Command OK\r\n`, e.g. `sw i03 Command OK\r\n`, within about 20 ms |
| Rejection | The command echoed back followed by ` Command incorrect\r\n` |
| Output on/off | `sw on`, `sw off` |
| Switch mode | `swmode next`, `swmode i0N priority`, `swmode off`, `swmode pod on|off` |
| Read settings | `read` - returns six lines, see below |

Captured from the real unit (firmware V1.1.104), the `read` reply is:

```
read Command OK\r\n
Input: port  1\r\n
Output: ON\r\r\n
Mode: NEXT\r\r\n
Pod: OFF\r\r\n
F/W: V1.1.104\r\r\n
```

Two quirks: there are two spaces before the port number, and lines after the first two end in `\r\r\n`. The switch sends the whole reply fast enough that several lines arrive in one UART chunk, so the parser splits on `\n` and trims trailing `\r` rather than assuming one line per callback.

The switch's RS232 port is a female DB9 wired DCE: it transmits on pin 2 and receives on pin 3. A USB-to-serial adapter with a male DB9 is a DTE and plugs straight in. The HW-044 module is also female and also transmits on pin 2, so module to switch needs a **male-to-male null modem** cable. A straight cable puts two transmitters on the same pin and nothing is received, which looks exactly like a dead switch.

## Wiring

| Module pin | ESP32 |
|---|---|
| VCC | 3V3 |
| GND | GND |
| TXD | GPIO17, UART2 TX |
| RXD | GPIO16, UART2 RX |

GPIO17 and 16 are UART2's defaults on a classic ESP32 and leave UART0 free for USB logging. Module labelling varies by vendor; on most, `TXD` is the TTL input that goes out of the DB9, so it connects to the ESP's TX. If nothing comes back, swap them.

LEDs: green on GPIO25 through 220 Ω, red on GPIO26 through 330 Ω, long legs to the GPIO side, short legs to GND. Those are 2 to 2.2 V, 20 mA parts running at about 6 and 4 mA; on the bench the red looked best at half the green's brightness, so the config drives red at 50% PWM. Wires are 26 AWG silicone stranded, soldered to the devkit's pin tails so the board can mount upside down with nothing above it.

**Bench test without the switch:** short DB9 pins 2 and 3 on the module. The 30-second `read` poll comes straight back and shows in `Last Switch Response`.

## ESPHome config

[`hdmi-switch.yaml`](hdmi-switch.yaml) is the source of truth. In outline:

- `uart` on GPIO17/GPIO16 at 19200. A debug sequence receives each chunk, splits it into lines, records the time of every reply, publishes each line to the `Last Switch Response` text sensor, and sets `current_input` from either a `sw i0N Command OK` echo or the `Input: port N` line of a `read` reply.
- Four template `switch` entities, one per input, each on only while that input is active. `turn_on_action` runs the `select_input` script, which writes the `sw` command, records the time so the poll holds off for 3 s, and resends once with a log warning if `current_input` hasn't changed within 600 ms. There is no `turn_off_action`.
- An `interval` sends `read` every 30 s unless a command went out in the last 3 s.
- `Switch Responding`, a connectivity binary sensor, is false when no reply of any kind has arrived for 90 s.
- Two `ledc` PWM outputs as internal monochromatic lights, refreshed twice a second: green at 100% when the API is connected, toggling when not; red at 50% when the switch isn't responding. The percentages are in the `refresh_leds` script.
- A `Restart` button.

ESPHome removed custom components in 2025.2; the UART debug sequence is the built-in way to read lines without one. [ssieb's `serial` text sensor](https://github.com/ssieb/esphome_components) is the external-component alternative if the debug hook ever misbehaves.

## Home Assistant

Native ESPHome API, no MQTT. The four switches map to a `buttons` row in an entities card, or four button cards in a grid; the built-in button card highlights an on switch, so the active input lights up with no custom cards. Core HA can't render a `select` entity as a button row (the tile card's `select-options` feature is a dropdown only), which is why the inputs are switches rather than a select.

## Case

Designed in Fusion 360 by Phil; not in this repo. Numbers that constrained it, all in millimetres:

- **DB9 cutout, flange through the wall:** the shell size E flange is 30.8 × 12.55, so a 32 × 14 rectangle centred on the shell centreline. The flange sits in the thickness of a 2 mm wall so the cable hood can seat. If instead the wall is kept in front of the flange, use the standard panel cutout: 19.74 wide side, 15.80 narrow side, 11.18 high, 10° sides, plus two Ø6 holes at 24.99 centres for the hex standoffs, and pocket the wall to about 1.2 around the D.
- **HW-044 module:** 29 × 27 PCB, two Ø3 holes at 26.5 centres 2 in from the rear edge, shell centreline about 7.6 above the PCB underside, pin tails need 3 under the board, nothing else under it. Mounting blocks were drilled in place with the boards fitted.
- **Devkit, inverted:** wires on the pin tails point up, USB-C sits near the floor, BOOT and EN face down. The PCB antenna points along the box toward the DB9 end; Wi-Fi was fine on the bench and in the cabinet.
- **LEDs:** Ø3.2 holes, pushed in from inside, the 3.8 flange stops them.
- **Cable hood clearance:** keep a flat 40 × 22 zone outside the DB9 for the plug's overmould and thumbscrews.
- **Heat-set M3 inserts:** Ø4.0 hole, insert length plus 1 deep, at least 2 of material around and below, so a boss of Ø8 or more.
- **Material:** PETG, 2 mm walls, 4 wall loops, 20% gyroid. Floor slots under the devkit and lid slots at both ends give the box a convection path, though it dissipates under a watt.

## Sources

- [VS481C manual on ManualsLib](https://www.manualslib.com/manual/1362150/Aten-Vs481c.html), [VS481B manual](https://www.manualslib.com/manual/936392/Aten-Vs481b.html), [ATEN CLI guide](https://manuals.plus/aten/command-line-interface-control-system-manual)
- [Spectrum Control D-sub board and panel cutouts](https://www.spectrumcontrol.com/asset/d-sub-board-panel-cutouts.pdf)
- [ESPHome UART component](https://esphome.io/components/uart.html), [UART read without a custom component](https://community.home-assistant.io/t/how-to-uart-read-without-custom-component/491950), [ESPHome: removal of custom components](https://developers.esphome.io/blog/2025/02/19/about-the-removal-of-support-for-custom-components/)
- [HA card features](https://www.home-assistant.io/dashboards/features/), [select-options icon style discussion](https://github.com/home-assistant/frontend/discussions/23888)
