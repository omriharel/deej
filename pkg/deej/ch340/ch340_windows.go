//go:build windows && amd64

// Package ch340 talks to a WCH CH340/CH341 USB-serial chip straight through
// Microsoft's WinUSB driver, in user mode. The vendor kernel driver
// (CH341S64.SYS), which can bluescreen Windows, is never loaded.
//
// The chip has to be bound to WinUSB first (e.g. with Zadig). See docs/winusb.md.
//
// The CH340 protocol (vendor control requests + one bulk IN endpoint) is the
// same one the Linux ch341 driver and Android's usb-serial-for-android use.
package ch340

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Hardware IDs of the WCH USB-serial chips we know how to drive.
var supportedIDs = []string{
	`VID_1A86&PID_7523`, // CH340
	`VID_1A86&PID_5523`, // CH341 in serial mode
}

// GUID_DEVINTERFACE_USB_DEVICE: registered by the USB hub for every device,
// whatever its function driver is. With WinUSB as the function driver,
// opening this path and calling WinUsb_Initialize works.
var guidUSBDevice = windows.GUID{Data1: 0xA5DCBF10, Data2: 0x6530, Data3: 0x11D2,
	Data4: [8]byte{0x90, 0x1F, 0x00, 0xC0, 0x4F, 0xB9, 0x51, 0xED}}

var (
	// ErrNotFound means no CH340 is plugged in.
	ErrNotFound = errors.New("no CH340/CH341 USB-serial device is plugged in")
	// ErrWrongDriver means a CH340 is plugged in but is not bound to WinUSB.
	ErrWrongDriver = errors.New("CH340 is not using the WinUSB driver")
	// ErrClosed is returned by Read/Write after Close.
	ErrClosed = errors.New("ch340: device closed")
)

// DeviceInfo describes a plugged-in CH340 as the PnP manager sees it.
// Collecting it only reads PnP metadata; it never talks to the device or its driver.
type DeviceInfo struct {
	InstanceID  string
	Description string
	Service     string   // bound driver service, e.g. "WinUSB" or "CH341SER_A64"
	Paths       []string // device interface paths we could open
}

// UsesWinUSB reports whether the device is bound to Microsoft's WinUSB driver.
func (d DeviceInfo) UsesWinUSB() bool { return strings.EqualFold(d.Service, "WinUSB") }

// Find lists every plugged-in CH340/CH341.
func Find() ([]DeviceInfo, error) {
	set, err := windows.SetupDiGetClassDevsEx(nil, "USB", 0,
		windows.DIGCF_PRESENT|windows.DIGCF_ALLCLASSES, 0, "")
	if err != nil {
		return nil, fmt.Errorf("enumerate USB devices: %w", err)
	}
	defer set.Close()

	var found []DeviceInfo
	for i := 0; ; i++ {
		data, err := set.EnumDeviceInfo(i)
		if err != nil {
			if errors.Is(err, windows.ERROR_NO_MORE_ITEMS) {
				break
			}
			continue
		}

		id, err := set.DeviceInstanceID(data)
		if err != nil || !isSupported(id) {
			continue
		}

		info := DeviceInfo{InstanceID: id}
		info.Service = regString(set, data, windows.SPDRP_SERVICE)
		info.Description = regString(set, data, windows.SPDRP_FRIENDLYNAME)
		if info.Description == "" {
			info.Description = regString(set, data, windows.SPDRP_DEVICEDESC)
		}

		// Zadig/libwdi installs write a custom interface GUID; prefer that path.
		for _, g := range customInterfaceGUIDs(set, data) {
			if paths, err := windows.CM_Get_Device_Interface_List(id, &g,
				windows.CM_GET_DEVICE_INTERFACE_LIST_PRESENT); err == nil {
				info.Paths = append(info.Paths, nonEmpty(paths)...)
			}
		}
		if paths, err := windows.CM_Get_Device_Interface_List(id, &guidUSBDevice,
			windows.CM_GET_DEVICE_INTERFACE_LIST_PRESENT); err == nil {
			info.Paths = append(info.Paths, nonEmpty(paths)...)
		}

		found = append(found, info)
	}
	return found, nil
}

func isSupported(instanceID string) bool {
	up := strings.ToUpper(instanceID)
	for _, s := range supportedIDs {
		if strings.Contains(up, s) {
			return true
		}
	}
	return false
}

func regString(set windows.DevInfo, data *windows.DevInfoData, prop windows.SPDRP) string {
	v, err := set.DeviceRegistryProperty(data, prop)
	if err != nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case []string:
		return strings.Join(t, ";")
	}
	return ""
}

func customInterfaceGUIDs(set windows.DevInfo, data *windows.DevInfoData) []windows.GUID {
	h, err := set.OpenDevRegKey(data, windows.DICS_FLAG_GLOBAL, 0, windows.DIREG_DEV, windows.KEY_READ)
	if err != nil {
		return nil
	}
	k := registry.Key(h)
	defer k.Close()

	var raw []string
	if list, _, err := k.GetStringsValue("DeviceInterfaceGUIDs"); err == nil {
		raw = append(raw, list...)
	}
	if one, _, err := k.GetStringValue("DeviceInterfaceGUID"); err == nil {
		raw = append(raw, one)
	}

	var out []windows.GUID
	for _, s := range raw {
		if g, err := windows.GUIDFromString(strings.TrimSpace(s)); err == nil {
			out = append(out, g)
		}
	}
	return out
}

func nonEmpty(in []string) []string {
	var out []string
	for _, s := range in {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// ---------------------------------------------------------------- WinUSB calls

var (
	modWinUSB                  = windows.NewLazySystemDLL("winusb.dll")
	procInitialize             = modWinUSB.NewProc("WinUsb_Initialize")
	procFree                   = modWinUSB.NewProc("WinUsb_Free")
	procControlTransfer        = modWinUSB.NewProc("WinUsb_ControlTransfer")
	procReadPipe               = modWinUSB.NewProc("WinUsb_ReadPipe")
	procWritePipe              = modWinUSB.NewProc("WinUsb_WritePipe")
	procAbortPipe              = modWinUSB.NewProc("WinUsb_AbortPipe")
	procFlushPipe              = modWinUSB.NewProc("WinUsb_FlushPipe")
	procSetPipePolicy          = modWinUSB.NewProc("WinUsb_SetPipePolicy")
	procQueryInterfaceSettings = modWinUSB.NewProc("WinUsb_QueryInterfaceSettings")
	procQueryPipe              = modWinUSB.NewProc("WinUsb_QueryPipe")
)

const (
	pipeTransferTimeout = 0x03 // WinUSB pipe policy, value in ms
	usbdPipeTypeBulk    = 2

	errSemTimeout       = syscall.Errno(121) // ERROR_SEM_TIMEOUT: read timed out, no data
	errOperationAborted = syscall.Errno(995) // ERROR_OPERATION_ABORTED: pipe aborted by Close

	readTimeoutMs = 500

	reqTypeVendorIn  = 0xC0
	reqTypeVendorOut = 0x40

	reqReadVersion = 0x5F
	reqReadReg     = 0x95
	reqWriteReg    = 0x9A
	reqSerialInit  = 0xA1
	reqModemCtrl   = 0xA4
)

type pipeInformation struct {
	PipeType          int32
	PipeID            uint8
	_                 uint8
	MaximumPacketSize uint16
	Interval          uint8
	_                 [3]uint8
}

// Device is an open CH340, usable as an io.ReadWriteCloser.
type Device struct {
	Info    DeviceInfo
	Version uint16 // chip version reported by the CH340

	file     windows.Handle
	usb      uintptr // WINUSB_INTERFACE_HANDLE
	pipeIn   uint8
	pipeOut  uint8
	inPacket int

	ioMu    sync.Mutex // held while a WinUSB call is in flight
	closed  atomic.Bool
	buf     []byte
	pending []byte
}

// Open finds the first CH340 bound to WinUSB and configures it for 8N1 at baud.
// It refuses to touch a CH340 that is still bound to the vendor driver.
func Open(baud int) (*Device, error) {
	devs, err := Find()
	if err != nil {
		return nil, err
	}
	if len(devs) == 0 {
		return nil, ErrNotFound
	}

	var target *DeviceInfo
	for i := range devs {
		if devs[i].UsesWinUSB() {
			target = &devs[i]
			break
		}
	}
	if target == nil {
		svc := devs[0].Service
		if svc == "" {
			svc = "no driver"
		}
		return nil, fmt.Errorf("%w (%s is bound to %q)", ErrWrongDriver, devs[0].InstanceID, svc)
	}
	if len(target.Paths) == 0 {
		return nil, fmt.Errorf("CH340 %s uses WinUSB but exposes no device interface", target.InstanceID)
	}

	var lastErr error
	for _, path := range target.Paths {
		d, err := openPath(path)
		if err != nil {
			lastErr = err
			continue
		}
		d.Info = *target
		if err := d.configure(baud); err != nil {
			d.Close()
			return nil, fmt.Errorf("configure CH340: %w", err)
		}
		return d, nil
	}
	return nil, fmt.Errorf("open CH340 via WinUSB: %w", lastErr)
}

func openPath(path string) (*Device, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(p,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil, windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OVERLAPPED, 0)
	if err != nil {
		return nil, fmt.Errorf("CreateFile: %w", err)
	}

	var usb uintptr
	if r, _, e := procInitialize.Call(uintptr(h), uintptr(unsafe.Pointer(&usb))); r == 0 {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("WinUsb_Initialize: %w", e)
	}

	d := &Device{file: h, usb: usb, pipeIn: 0x82, pipeOut: 0x02, inPacket: 32}
	d.discoverPipes()
	// Read one USB packet per ReadPipe. A bigger request only completes once it
	// is full (the CH340 sends full 32-byte packets back to back), and if the
	// pipe timeout cancels it first, whatever it had collected is lost.
	d.buf = make([]byte, d.inPacket)
	return d, nil
}

// discoverPipes reads the bulk endpoints from the interface descriptor; the
// defaults (0x82 in, 0x02 out) are what every CH340 uses anyway.
func (d *Device) discoverPipes() {
	var desc [9]byte // USB_INTERFACE_DESCRIPTOR
	if r, _, _ := procQueryInterfaceSettings.Call(d.usb, 0, uintptr(unsafe.Pointer(&desc[0]))); r == 0 {
		return
	}
	numEndpoints := int(desc[4])
	for i := 0; i < numEndpoints; i++ {
		var pi pipeInformation
		if r, _, _ := procQueryPipe.Call(d.usb, 0, uintptr(i), uintptr(unsafe.Pointer(&pi))); r == 0 {
			continue
		}
		if pi.PipeType != usbdPipeTypeBulk {
			continue
		}
		if pi.PipeID&0x80 != 0 {
			d.pipeIn = pi.PipeID
			if pi.MaximumPacketSize > 0 {
				d.inPacket = int(pi.MaximumPacketSize)
			}
		} else {
			d.pipeOut = pi.PipeID
		}
	}
}

// control sends a vendor control request. WINUSB_SETUP_PACKET is 8 bytes and is
// passed by value, which on amd64 means packed into one register.
func (d *Device) control(reqType, req uint8, value, index uint16, data []byte) (int, error) {
	setup := uint64(reqType) | uint64(req)<<8 | uint64(value)<<16 |
		uint64(index)<<32 | uint64(len(data))<<48
	var ptr *byte
	if len(data) > 0 {
		ptr = &data[0]
	}
	var n uint32
	r, _, e := procControlTransfer.Call(d.usb, uintptr(setup), uintptr(unsafe.Pointer(ptr)),
		uintptr(len(data)), uintptr(unsafe.Pointer(&n)), 0)
	if r == 0 {
		return 0, fmt.Errorf("control 0x%02X/0x%02X: %w", reqType, req, e)
	}
	return int(n), nil
}

func (d *Device) out(req uint8, value, index uint16) error {
	_, err := d.control(reqTypeVendorOut, req, value, index, nil)
	return err
}

func (d *Device) in(req uint8, value, index uint16) ([]byte, error) {
	b := make([]byte, 2)
	n, err := d.control(reqTypeVendorIn, req, value, index, b)
	return b[:n], err
}

// Tweaks for diagnosing with ch340probe; deej uses the defaults.
var (
	// AssertDTR raises DTR/RTS on open like a normal COM port would (this resets an Arduino).
	AssertDTR = false
	// LinuxInit uses the Linux ch341 driver's shorter init sequence instead of Android's.
	LinuxInit = false
)

// configure runs the same init sequence as usb-serial-for-android's CH34x driver.
// It never asserts DTR, so the Arduino is not reset on connect.
func (d *Device) configure(baud int) error {
	if v, err := d.in(reqReadVersion, 0, 0); err != nil {
		return err
	} else if len(v) == 2 {
		d.Version = uint16(v[0])<<8 | uint16(v[1])
	}

	modem := uint16(0xFFFF) // DTR and RTS off
	if AssertDTR {
		modem = ^uint16(0x20|0x40) & 0xFF
	}

	steps := []func() error{
		func() error { return d.out(reqSerialInit, 0, 0) },
		func() error { return d.setBaud(baud) },
		func() error { _, err := d.in(reqReadReg, 0x2518, 0); return err },
		func() error { return d.out(reqWriteReg, 0x2518, 0x00C3) }, // LCR: RX+TX on, 8 data bits, no parity, 1 stop
		func() error { _, err := d.in(reqReadReg, 0x0706, 0); return err },
		func() error { return d.out(reqSerialInit, 0x501F, 0xD90A) },
		func() error { return d.setBaud(baud) },
		func() error { return d.out(reqModemCtrl, modem, 0) },
		func() error { _, err := d.in(reqReadReg, 0x0706, 0); return err },
	}
	if LinuxInit {
		steps = []func() error{
			func() error { return d.out(reqSerialInit, 0, 0) },
			func() error { return d.setBaud(baud) },
			func() error { return d.out(reqWriteReg, 0x2518, 0x00C3) },
			func() error { return d.out(reqModemCtrl, modem, 0) },
		}
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}

	timeout := uint32(readTimeoutMs)
	if r, _, e := procSetPipePolicy.Call(d.usb, uintptr(d.pipeIn), pipeTransferTimeout,
		unsafe.Sizeof(timeout), uintptr(unsafe.Pointer(&timeout))); r == 0 {
		return fmt.Errorf("set read timeout: %w", e)
	}
	procFlushPipe.Call(d.usb, uintptr(d.pipeIn))
	return nil
}

func (d *Device) setBaud(baud int) error {
	if baud <= 0 {
		return fmt.Errorf("invalid baud rate %d", baud)
	}
	const baseFactor = 1532620800
	factor := int64(baseFactor / baud)
	divisor := int64(3)
	for factor > 0xfff0 && divisor > 0 {
		factor >>= 3
		divisor--
	}
	if factor > 0xfff0 {
		return fmt.Errorf("baud rate %d is too low for the CH340", baud)
	}
	factor = 0x10000 - factor
	divisor |= 0x80 // otherwise the chip waits for its buffer to fill before sending
	if err := d.out(reqWriteReg, 0x1312, uint16((factor&0xff00)|divisor)); err != nil {
		return err
	}
	return d.out(reqWriteReg, 0x0F2C, uint16(factor&0xff))
}

// Read blocks until serial data arrives, the device goes away, or Close is called.
func (d *Device) Read(p []byte) (int, error) {
	if len(d.pending) > 0 {
		n := copy(p, d.pending)
		d.pending = d.pending[n:]
		return n, nil
	}
	for {
		if d.closed.Load() {
			return 0, ErrClosed
		}
		d.ioMu.Lock()
		if d.closed.Load() {
			d.ioMu.Unlock()
			return 0, ErrClosed
		}
		var n uint32
		r, _, e := procReadPipe.Call(d.usb, uintptr(d.pipeIn), uintptr(unsafe.Pointer(&d.buf[0])),
			uintptr(len(d.buf)), uintptr(unsafe.Pointer(&n)), 0)
		d.ioMu.Unlock()

		if r == 0 {
			if d.closed.Load() || e == errOperationAborted {
				return 0, ErrClosed
			}
			if e != errSemTimeout {
				return 0, fmt.Errorf("WinUsb_ReadPipe: %w", e)
			}
			// timed out: keep anything that did arrive, otherwise keep waiting
		}
		if n == 0 {
			continue
		}
		c := copy(p, d.buf[:n])
		if c < int(n) {
			d.pending = append(d.pending[:0], d.buf[c:n]...)
		}
		return c, nil
	}
}

// Write sends bytes to the board (deej never does, but io.ReadWriteCloser wants it).
func (d *Device) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	d.ioMu.Lock()
	defer d.ioMu.Unlock()
	if d.closed.Load() {
		return 0, ErrClosed
	}
	var n uint32
	r, _, e := procWritePipe.Call(d.usb, uintptr(d.pipeOut), uintptr(unsafe.Pointer(&p[0])),
		uintptr(len(p)), uintptr(unsafe.Pointer(&n)), 0)
	if r == 0 {
		return int(n), fmt.Errorf("WinUsb_WritePipe: %w", e)
	}
	return int(n), nil
}

// Close aborts any pending read, then releases WinUSB and the file handle.
// Safe to call more than once.
func (d *Device) Close() error {
	if d.closed.Swap(true) {
		return nil
	}
	procAbortPipe.Call(d.usb, uintptr(d.pipeIn)) // wakes a blocked ReadPipe
	d.ioMu.Lock()                                // wait for it to return
	defer d.ioMu.Unlock()
	procFree.Call(d.usb)
	return windows.CloseHandle(d.file)
}

// String keeps zap's "conn" log field readable.
func (d *Device) String() string {
	return fmt.Sprintf("CH340 v%04X via WinUSB (%s, in 0x%02X, out 0x%02X)", d.Version, d.Info.InstanceID, d.pipeIn, d.pipeOut)
}
