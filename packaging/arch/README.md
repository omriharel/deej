# Arch Linux package

This directory builds `deej-git`, a pacman-managed build of the Linux fork.

## Build and install

```sh
cd packaging/arch
makepkg -si
```

`makepkg -si` asks for your sudo password to install the package.

Then enable the user service once:

```sh
systemctl --user daemon-reload
systemctl --user enable --now deej.service
```

The service starts with the systemd user manager and is ordered after the
graphical session and PipeWire. It also restarts automatically if the board or
audio server is not available yet.

## Configuration

On first start, `/usr/bin/deej` creates:

```text
~/.config/deej/config.yaml
```

The default config is installed at `/usr/share/deej/config.yaml`. Edit the file
in your home directory; it will not be overwritten on upgrades.

The package installs:

- `/usr/bin/deej` - launcher that uses `~/.config/deej`
- `/usr/lib/deej/deej` - the actual binary
- `/usr/lib/systemd/user/deej.service` - systemd user service
- `/usr/share/deej/config.yaml` - default config template

## Useful commands

```sh
systemctl --user status deej.service
systemctl --user restart deej.service
journalctl --user -u deej.service -f
pacman -Q deej-git
pacman -Rns deej-git
```
