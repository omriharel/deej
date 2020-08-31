package util

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"
	"github.com/mitchellh/go-ps"
)

const (
	getCurrentWindowInternalCooldown = time.Millisecond * 200
	uwpContainerName                 = "ApplicationFrameHost.exe"
)

var (
	lastGetCurrentWindowResult = ""
	lastGetCurrentWindowCall   = time.Now()
)

func getCurrentWindowProcessName() (string, error) {

	// apply an internal cooldown on this function to avoid calling windows API functions too frequently.
	// return a cached value during that cooldown
	now := time.Now()
	if lastGetCurrentWindowCall.Add(getCurrentWindowInternalCooldown).After(now) {
		return lastGetCurrentWindowResult, nil
	}

	lastGetCurrentWindowCall = now

	// the logic of this implementation is a bit convoluted because of the way UWP apps
	// (also known as "modern win 10 apps" or "microsoft store apps") work.
	// these are rendered in a parent container by the name of ApplicationFrameHost.exe.
	// when windows's GetForegroundWindow is called, it returns the window owned by that parent process.
	// so whenever we get that, we need to go and look through its child windows until we find one with a different PID

	// a struct holding parent and child PIDs for a given window thread
	type windowInfo struct {
		ownerPID uint32
		childPID uint32
	}

	// TODO: check for system PID (0)

	// a callback that will be called for each child window belonging to ApplicationFrameHost.exe
	enumChildWindowsCallback := func(childHWND *uintptr, lParam *uintptr) uintptr {

		// cast the outer lp into something we can work with (maybe closures are good enough?)
		info := (*windowInfo)(unsafe.Pointer(lParam))

		// get the child window's real PID
		var childPID uint32
		win.GetWindowThreadProcessId((win.HWND)(unsafe.Pointer(childHWND)), &childPID)

		// compare it to the parent's - if they're different, that's the one
		if childPID != info.ownerPID {
			info.childPID = childPID

			// we found the missing child, stop iterating
			return 0
		}

		// indicates to the system to keep iterating
		return 1
	}

	// get the current foreground window
	hwnd := win.GetForegroundWindow()
	info := windowInfo{}

	// get its PID and put it in our window info struct
	win.GetWindowThreadProcessId(hwnd, &info.ownerPID)

	// find the process name corresponding to the parent PID
	process, err := ps.FindProcess(int(info.ownerPID))
	if err != nil {
		return "", fmt.Errorf("get parent process for pid %d: %w", info.ownerPID, err)
	}

	result := process.Executable()

	// if it is a UWP app, iterate its child windows to attempt finding the real process
	if result == uwpContainerName {
		win.EnumChildWindows(hwnd, syscall.NewCallback(enumChildWindowsCallback), (uintptr)(unsafe.Pointer(&info)))

		actualProcess, err := ps.FindProcess(int(info.childPID))
		if err != nil {
			return "", fmt.Errorf("get child process for pid %d: %w", info.childPID, err)
		}

		result = actualProcess.Executable()
	}

	// return whichever executable name we ended up with
	lastGetCurrentWindowResult = result
	return result, nil
}
