# WinUSB mode for CH340 boards (Windows)

Many cheap Arduino Nano/Uno clones use a WCH **CH340** USB-serial chip, and on
Windows that means WCH's kernel driver, `CH341S64.SYS`. That driver can
bluescreen the PC while deej is reading from it (see issue #162). Crash dumps
show `IRQL_NOT_LESS_OR_EQUAL` / `DRIVER_IRQL_NOT_LESS_OR_EQUAL` inside
`CH341S64.SYS`, during a timer callback in the driver itself.

deej's own code runs in user mode and can't cause a bluescreen; the crash is in
the driver. WinUSB mode removes that driver from the picture:

- The board is switched to **WinUSB**, Microsoft's generic signed USB driver.
- deej talks to the CH340 chip directly through WinUSB, from user mode, using the
  same protocol as the Linux `ch341` driver and Android's usb-serial-for-android.
- `CH341S64.SYS` is no longer loaded for the board, so it can't crash the PC.

Your Arduino sketch doesn't change.

## Is this for me?

Only if your board shows up in Device Manager as **USB-SERIAL CH340** (USB ID
`1A86:7523`, or `1A86:5523` for a CH341). Boards with an FTDI, CP210x or
native-USB chip (Leonardo, Pro Micro, genuine Uno R3) don't use this driver and
don't need this.

## Setup

1. **Quit deej**, so the COM port isn't open while you change its driver.
2. Download **Zadig** from <https://zadig.akeo.ie> and run it.
3. **Options → List All Devices**. Pick the board. Zadig shows the chip's own
   USB name, usually **USB Serial**. Check that the USB ID is `1A86 7523` and the
   current driver is `CH341SER_A64`.
4. Select **WinUSB** as the target driver and click **Replace Driver**.
5. In `config.yaml` set:
   ```yaml
   com_port: winusb
   ```
   Keep `baud_rate` matching your sketch (9600 by default).
6. Start deej.

In WinUSB mode deej waits for the board if it isn't plugged in yet, and
reconnects by itself if you unplug and replug it.

To check the board without deej, use `ch340probe`. It's built from
`tools/ch340probe`; quit deej before running it.

```
ch340probe            # shows which driver the CH340 is using
ch340probe -read 5s   # prints the slider lines it receives over WinUSB
```

## Going back to the COM port

Do this to upload a new sketch from the Arduino IDE, or to undo the change.

1. Quit deej.
2. Device Manager → **Universal Serial Bus devices** → the board → **Uninstall device**,
   and tick *Attempt to remove the driver for this device*. This removes only
   Zadig's WinUSB package.
3. Unplug and replug the board. Windows reinstalls the CH340 driver and the COM
   port comes back. The COM number may change.
4. Set `com_port` back to that COM port.

## Notes

- Zadig matches by USB ID, so *other* CH340 gadgets you plug in later may also
  get WinUSB. Give those back their COM port with Device Manager → Update driver
  → Browse → Let me pick → *USB-SERIAL CH340*.
- Windows Update can occasionally put the CH340 driver back. deej then shows a
  notification saying the board isn't on WinUSB; run Zadig again.
- WinUSB mode is Windows (64-bit) only and supports CH340 and CH341 chips. Other
  `com_port` values work exactly as before.
- Tested so far on one CH340 (chip version 0x31) with the stock deej sketch at
  9600 baud.
