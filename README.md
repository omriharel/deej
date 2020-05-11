# deej

Arduino and Go project for controlling application volumes on Windows and Linux PCs with physical sliders (like a DJ!)

**_New:_ `deej` has been re-written as a Go application! [More details](#whats-new) | [Download](https://github.com/omriharel/deej/releases)**

**_New:_ join the [deej Discord server](https://discord.gg/nf88NJu) if you need help or have any questions!**

[![Discord](https://img.shields.io/discord/702940502038937667?logo=discord)](https://discord.gg/nf88NJu)

[**Video demonstration on YouTube**](https://youtu.be/VoByJ4USMr8)

[**See some awesome versions built by people around the world!**](./community.md)

![Physical build](assets/build.jpg)

## Table of contents

- [What's new](#whats-new)
- [How it works](#how-it-works)
  - [Hardware](#hardware)
    - [Schematic](#schematic)
  - [Software](#software)
- [Slider mapping (configuration)](#slider-mapping-configuration)
- [Build your own!](#build-your-own)
  - [Bill of Materials](#bill-of-materials)
  - [Build procedure](#build-procedure)
- [How to run](#how-to-run)
  - [Requirements](#requirements)
  - [Download and installation](#download-and-installation)
  - [Building from source](#building-from-source)
- [Community](#community)
- [Long-ish term roadmap](#long-ish-term-roadmap)
- [License](#license)

## What's new

`deej` is now written in Go, and [distributed](https://github.com/omriharel/deej/releases) as a single Windows executable (of course you can still [build from source](#building-from-source) if that's your thing).

This means you no longer have to maintain a Python environment. You can even build one for your friends, give them a simple download link and they'll be good to go!

In addition, check out these features:

- **Fully backwards-compatible** with your existing `config.yaml` and Arduino sketch
- **Faster** and more lightweight, consuming around 10MB of memory
- Runs from your system tray
- **Helpful notifications** will let you know if something isn't working
- New `system` flag lets you assign the "system sounds" volume level
- Supports everything the Python version did

> **Migrating from the Python version?** Great! You only need to keep your `config.yaml` file. Download the executable from the [releases page](https://github.com/omriharel/deej/releases/latest), place it alongside the configuration file and you're done.

> **Prefer to stick with Python?** That's totally fine. It will no longer be maintained, but you can always find it in the [`legacy-python` branch](https://github.com/omriharel/deej/tree/legacy-python).

## How it works

### Hardware

- The sliders are connected to 5 (or as many as you like) analog pins on an Arduino Nano/Uno board. They're powered from the board's 5V output (see schematic)
- The board connects via a USB cable to the PC

#### Schematic

![Hardware schematic](assets/schematic.png)

### Software

- The code running on the Arduino board is a C program constantly writing current slider values over its Serial interface [`deej-arduino.ino`](./deej-arduino.ino)
- The PC runs a lightweight Go client [`cmd/main.go`](./cmd/main.go) in the background. This client reads the serial stream and adjusts app volumes according to the given configuration file

## Slider mapping (configuration)

`deej` uses a simple YAML-formatted configuration file named [`config.yaml`](./config.yaml), placed alongside the deej executable.

The config file determines which applications are mapped to which sliders, and which COM port/baud rate to use for the connection to the Arduino board.

**This file auto-reloads when its contents are changed, so you can change application mappings on-the-fly without restarting `deej`.**

It looks like this:

```yaml
slider_mapping:
  0: master
  1: chrome.exe
  2: spotify.exe
  3:
    - pathofexile_x64.exe
    - rocketleague.exe
  4: discord.exe

# limits how often deej will look for new processes
process_refresh_frequency: 5

# settings for connecting to the arduino board
com_port: COM4
baud_rate: 9600
```
- `master` is a special option for controlling master volume of the system.
- _New:_ `system` is a special option for controlling the "System sounds" volume in the Windows mixer
- Process names aren't case-sensitive, meaning both `chrome.exe` and `CHROME.exe` will work
- You can create groups of process names (using a list) to either:
    - control more than one app with a single slider
    - choose whichever process in the group that's currently running (i.e. to have one slider control any game you're playing)

## Build your own!

Building `deej` is very simple. You only need a few cheap parts - it's an excellent starter project (and my first Arduino project, personally). Remember that if you need any help or have a question that's not answered here, you can always [join the deej Discord server](https://discord.gg/nf88NJu).

Build `deej` for yourself, or as an awesome gift for your gaming buddies!

### Bill of Materials

- An Arduino Nano or Uno board
  - I officially recommend using a Nano as it offers a smaller form-factor, a friendlier USB connector and more analog pins. Plus it's cheaper
- A few slider potentiometers, up to your number of free analog pins (they're around 1-2 USD each, and come with a standard 10K Ohm variable resistor)
  - **Important:** make sure to get **linear** sliders, not logarithmic ones! Check the product description
  - You can also use circular knobs if you like
- Some wires
- Any kind of box to hold everything together

### Build procedure

- Connect everything according to the [schematic](#schematic)
- Test with a multimeter to be sure your sliders are hooked up correctly
- Flash the Arduino chip with the sketch in [`arduino\deej-5-sliders-vanilla`](./arduino/deej-5-sliders-vanilla/deej-5-sliders-vanilla.ino)
  - If you have more or less than 5 sliders, you can edit the sketch to match what you have
- After flashing, check the serial monitor. You should see a constant stream of values separated by a pipe (`|`) character, e.g. `0|240|1023|0|483`
  - When you move a slider, its corresponding value should move between 0 and 1023
- Congratulations, you're now ready to run the `deej` executable!

## How to run

### Requirements

#### Windows

- Windows. That's it

#### Linux

- Install `libgtk-3-dev`, `libappindicator3-dev` and `libwebkit2gtk-4.0-dev` for system tray support

### Download and installation

- Head over to the [releases page](https://github.com/omriharel/deej/releases) and download the [latest version](https://github.com/omriharel/deej/releases/latest)'s executable and configuration file (`deej.exe` and `config.yaml`)
- Place them in the same directory anywhere on your machine
- (Optional) Create a shortcut to `deej.exe` and copy it to `%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup` to have `deej` run on boot

### Building from source

If you'd rather not download a compiled executable, or want to extend `deej` or modify it to your needs, feel free to clone the repository and build it yourself. All you need is a somewhat recent (v1.12-ish+) Go environment on your machine.

Like other Go packages, you can also use the `go get` tool: `go get -u github.com/omriharel/deej`.

If you need any help with this, please [join our Discord server](https://discord.gg/nf88NJu).

## Community

[![Discord](https://img.shields.io/discord/702940502038937667?logo=discord)](https://discord.gg/nf88NJu)

While `deej` is still a very new project, a vibrant community has already started to grow around it. Come hang out with us in the [deej Discord server](https://discord.gg/nf88NJu), or check out awesome builds made by our members in the [community showcase](./community.md).

## Long-ish term roadmap

- Serial communications rework to support two-way data flows for better extensibility
- Mic input support
- Basic GUI to replace manual configuration editing
- Feel free to open an issue if you feel like something else is missing

## License

`deej` is released under the [MIT license](./LICENSE).
