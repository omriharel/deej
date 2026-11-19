//go:build !(windows && amd64)

package ch340

import "errors"

var (
	ErrNotFound    = errors.New("no CH340/CH341 USB-serial device is plugged in")
	ErrWrongDriver = errors.New("CH340 is not using the WinUSB driver")
	ErrClosed      = errors.New("ch340: device closed")
)

var errUnsupported = errors.New("WinUSB mode is only supported on 64-bit Windows")

type DeviceInfo struct {
	InstanceID, Description, Service string
	Paths                            []string
}

func (d DeviceInfo) UsesWinUSB() bool { return false }

func Find() ([]DeviceInfo, error) { return nil, errUnsupported }

type Device struct {
	Info    DeviceInfo
	Version uint16
}

func Open(baud int) (*Device, error)          { return nil, errUnsupported }
func (d *Device) Read(p []byte) (int, error)  { return 0, errUnsupported }
func (d *Device) Write(p []byte) (int, error) { return 0, errUnsupported }
func (d *Device) Close() error                { return nil }
