# deej

Arduino project for controlling audio volume for separate Windows processes using physical sliders (like a DJ!)

**_New:_ join the [deej Discord server](https://discord.gg/nf88NJu) if you need help or have any questions!**

[![Discord](https://img.shields.io/discord/702940502038937667?logo=discord)](https://discord.gg/nf88NJu)

[**Video demonstration on YouTube**](https://youtu.be/VoByJ4USMr8)

[**See some awesome versions built by people around the world!**](./community.md)

![Physical build](assets/build.jpg)

## How it works

### Hardware

- The sliders are connected to 5 analog pins on an Arduino Nano/Uno board. They're powered from the board's 5V output (see schematic)
- The board connects via a USB cable to the PC

#### Schematic

![Hardware schematic](assets/schematic.png)

### Software

- The code running on the Arduino board is a C program constantly writing current slider values over its Serial interface [`deej-arduino.ino`](./deej-arduino.ino)
- The PC runs a Python script [`deej.py`](./deej.py) that listens to the board's Serial connection, detects changes in slider values and sets volume of equivalent audio sessions accordingly.
- A VBScript-based run helper [`run.vbs`](./run.vbs) allows this Python script to run in the background (from the Windows tray).

## Slider mapping (configuration)

`deej` uses an external YAML-formatted configuration file named [`config.yaml`](./config.yaml).

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

# recommend to leave this setting at its default value
process_refresh_frequency: 5

# settings for connecting to the arduino board
com_port: COM4
baud_rate: 9600
```

- Process names aren't case-sensitive
- You can use a list of process names to either:
    - define a group that is controlled simultaneously
    - choose whichever process in the group is currently running (in this example, one slider is for different games that may be running)
- `master` is a special option for controlling master volume of the system.
- The `process_refresh_frequency` option limits how often `deej` may look for new processes if their appropriate slider moves. This allows you to leave `deej` running in background and open/close processes normally - the sliders will #justwork

## Build your own

Building `deej` is very simple! You only need a few cheap parts - it's an excellent starter project (and my first Arduino project, personally). Remember that if you need any help or have a question that's not answered here, you can always [join the deej Discord server](https://discord.gg/nf88NJu).

### Bill of Materials

- An Arduino Nano or Uno board - I officially recommend using a Nano as it offers a smaller form-factor, a friendlier USB connector and more analog pins. Plus it's cheaper
- As many slider potentiometers as you like, up to your number of analog pins (they're around 1-2 USD each, and come with a standard 10K Ohm variable resistor)
  - **Important:** make sure to get linear sliders, not logarithmic ones!
  - You can also use circular knobs if you like (but then you're not a DJ!)
- Some wires and a way to connect them to your chosen board and sliders. This can be done with or without a breadboard, depending on how you're comfortable doing it
- Any kind of box to hold everything together. It can be anything from a simple shoebox to a well-designed 3D-printed enclosure for a very professional look

## How to run

If you've actually gone ahead and built it, here's how you can run `deej`:

### Requirements

- Python 2.7.x (Sorry!) and `pip`
- `virtualenv`

### Installation

- Download the repository by either cloning it or downloading its archive.
- In the repo's directory, run:
    - `virtualenv venv`
    - `venv\Scripts\activate.bat`
    - `pip install -r requirements.txt`
- Make a shortcut to `run.vbs` by right-clicking it -> "Create Shortcut"
- (Optional, but mandatory) Change the shortcut's icon to `assets/logo.ico`
- (Optional, but optional) Copy the shortcut to `%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup` to have `deej` run on boot

## Missing stuff

- Better logging and error handling
- Automatic COM port detection
- Mic input support
- Feel free to let me know if there's more demand for other features!
