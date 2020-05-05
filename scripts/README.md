## Developer scripts

This document lists the various scripts in the project and their purposes.

> Note: All scripts are meant to be run from the root of the repository, i.e. from the `deej` directory: `scripts\whatever.bat`. They're not guaranteed to work correctly if run from another directory.

- [`build-dev.bat`](./build-dev.bat): Builds deej with a console window, for development purposes
- [`build-release.bat`](./build-release.bat): Builds deej as a standalone tray application without a console window, for releases
- [`build-all.bat`](./build-all.bat): Helper script to build all variants
- [`make-icon.bat`](./make-icon.bat): Converts a .ico file to an icon byte array in a Go file. Used by our systray library. You shouldn't need to run this unless you change the deej logo
- [`make-rsrc.bat`](./make-rsrc.bat): Generates a `rsrc.syso` resource file inside `cmd` alongside `main.go` - This indicates to the Go linker to use the deej application manifest and icon when building.
