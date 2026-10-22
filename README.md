# ROG Delta II for Linux

[![CI](https://github.com/mbekkelund/rog-delta-ii/actions/workflows/ci.yml/badge.svg)](https://github.com/mbekkelund/rog-delta-ii/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A native Go control center for the ASUS ROG Delta II 2.4 GHz headset
(`0b05:1afa`). It provides a focused Linux desktop interface without Armoury Crate.

> [!NOTE]
> This is an independent community project. It is not affiliated with or endorsed by ASUS.

## Features

- Detects the correct headset HID interface without assuming a fixed `/dev/hidrawN` number.
- Controls PipeWire output volume and output/microphone mute.
- Includes Balanced, Movie, Gaming, Communication, Music, and Bass Boost profiles.
- Provides a custom ten-band equalizer with bass, mid, treble, and fine-grained controls.
- Provides adjustable low-latency software sidetone through PipeWire.
- Applies profiles through EasyEffects 8 with clipping-safe output headroom.
- Provides a responsive desktop UI and a complete CLI.
- Passively records timestamped HID reports for protocol research.
- Ships Debian packaging, a narrowly scoped udev rule, and a systemd user unit.

The battery level and charging state are read from vendor reports `cc 12 09 00 00 XX` (XX = percent) and
`cc 12 08 00 00 YY` (YY = 1 while charging). RGB, hardware sidetone, and hardware EQ remain discovery features
until their HID commands have been verified. The recorder never writes to the device.

## Requirements

- Linux with PipeWire and WirePlumber
- EasyEffects 8 or newer
- Vivaldi, Chromium, or Chrome for the dedicated desktop window
- Go 1.24 or newer when building from source

## Install on Debian or Ubuntu

Download the `.deb` from the latest GitHub release and install it with:

```sh
sudo apt install ./rogdelta_0.2.1_amd64.deb
```

Reconnect the USB dongle after installation so the udev access rule takes effect. ROG Delta II
will then be available in the desktop application menu.

## Build from source

```sh
git clone https://github.com/mbekkelund/rog-delta-ii.git
cd rog-delta-ii
make test build
./bin/rogdelta desktop
```

Build a local Debian package:

```sh
make deb
sudo apt install ./dist/rogdelta_0.1.0_amd64.deb
```

## CLI

```text
rogdelta desktop
rogdelta status
rogdelta volume 60
rogdelta mute
rogdelta mic-mute
rogdelta preset gaming
rogdelta record --seconds 15 --output capture.jsonl
```

Run `rogdelta help` for the complete command reference.

## Desktop implementation

The Go process serves the embedded interface on a random loopback port and opens it in an
isolated browser app window. Closing the window stops the private server and removes its
temporary profile. The user's regular browser profile is never accessed.

Volume, sidetone, selected profile, and custom equalizer settings are restored between sessions.
They are stored in `~/.config/rogdelta/settings.json` (or the equivalent XDG config directory).

## HID protocol research

The descriptor exposes output report `0x06`, input report `0x07`, and vendor input/output report
`0xCC`. Their payload semantics are not public. Unknown writes are deliberately disabled.

See [CONTRIBUTING.md](CONTRIBUTING.md) before sharing captures or implementing device writes.

## License

[MIT](LICENSE)

## About this fork

Fork of [mbekkelund/rog-delta-ii](https://github.com/mbekkelund/rog-delta-ii) with a compact window and a
battery icon in the system tray (`make install-tray`, then `rogdelta tray`). Not affiliated with ASUS.
