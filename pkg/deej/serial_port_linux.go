package deej

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type usbSerialDevice struct {
	name    string
	path    string
	byID    string
	vid     string
	pid     string
	product string
}

// resolveSerialPort returns the serial device to open. When com_port is
// "auto", it scans sysfs for USB serial adapters (CH340, CP210x, FTDI,
// Arduino, etc.) and returns a stable /dev/serial/by-id path when possible.
// If usb_vid/usb_pid are set, only a device with those identifiers matches.
func resolveSerialPort(configured, wantVID, wantPID string) (string, error) {
	configured = strings.TrimSpace(configured)
	if configured != "" && !strings.EqualFold(configured, autoCOMPort) {
		return configured, nil
	}

	devices, err := findUSBSerialDevices()
	if err != nil {
		return "", err
	}

	if len(devices) == 0 {
		return "", fmt.Errorf("no USB serial devices found")
	}

	wantVID = normalizeUSBID(wantVID)
	wantPID = normalizeUSBID(wantPID)

	matches := make([]usbSerialDevice, 0, len(devices))
	for _, device := range devices {
		if wantVID != "" && !strings.EqualFold(device.vid, wantVID) {
			continue
		}
		if wantPID != "" && !strings.EqualFold(device.pid, wantPID) {
			continue
		}

		// With no explicit USB filter, only consider known USB-serial chips.
		// This avoids accidentally opening an unrelated ttyUSB device.
		if wantVID == "" && wantPID == "" && !isKnownUSBSerial(device.vid, device.pid) {
			continue
		}

		matches = append(matches, device)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no matching USB serial device found; available devices: %s",
			describeUSBDevices(devices))
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("multiple USB serial devices match (%s); set com_port explicitly or use usb_vid/usb_pid",
			describeUSBDevices(matches))
	}

	if matches[0].byID != "" {
		return matches[0].byID, nil
	}

	return matches[0].path, nil
}

func findUSBSerialDevices() ([]usbSerialDevice, error) {
	entries, err := ioutil.ReadDir("/sys/class/tty")
	if err != nil {
		return nil, fmt.Errorf("read /sys/class/tty: %w", err)
	}

	byID := byIDPaths()
	devices := []usbSerialDevice{}

	for _, entry := range entries {
		name := entry.Name()
		if !looksLikeUSBSerial(name) {
			continue
		}

		devicePath := filepath.Join("/dev", name)
		if _, err := os.Stat(devicePath); err != nil {
			continue
		}

		usbDir := usbDeviceDir(filepath.Join("/sys/class/tty", name, "device"))
		if usbDir == "" {
			continue
		}

		vid := readSysfs(usbDir, "idVendor")
		pid := readSysfs(usbDir, "idProduct")
		if vid == "" || pid == "" {
			continue
		}

		devices = append(devices, usbSerialDevice{
			name:    name,
			path:    devicePath,
			byID:    byID[name],
			vid:     vid,
			pid:     pid,
			product: readSysfs(usbDir, "product"),
		})
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].name < devices[j].name
	})

	return devices, nil
}

func looksLikeUSBSerial(name string) bool {
	return strings.HasPrefix(name, "ttyUSB") ||
		strings.HasPrefix(name, "ttyACM") ||
		strings.HasPrefix(name, "ttyCH343") ||
		strings.HasPrefix(name, "ttyCH341")
}

// usbDeviceDir walks up from a tty device symlink until it finds the USB
// device directory containing idVendor/idProduct.
func usbDeviceDir(deviceLink string) string {
	resolved, err := filepath.EvalSymlinks(deviceLink)
	if err != nil {
		return ""
	}

	dir := resolved
	for depth := 0; depth < 6; depth++ {
		if readSysfs(dir, "idVendor") != "" && readSysfs(dir, "idProduct") != "" {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

func readSysfs(dir, name string) string {
	data, err := ioutil.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

func byIDPaths() map[string]string {
	result := map[string]string{}

	entries, err := ioutil.ReadDir("/dev/serial/by-id")
	if err != nil {
		return result
	}

	for _, entry := range entries {
		fullPath := filepath.Join("/dev/serial/by-id", entry.Name())
		resolved, err := filepath.EvalSymlinks(fullPath)
		if err != nil {
			continue
		}

		ttyName := filepath.Base(resolved)
		if _, exists := result[ttyName]; !exists {
			result[ttyName] = fullPath
		}
	}

	return result
}

func normalizeUSBID(id string) string {
	id = strings.TrimSpace(strings.ToLower(id))
	return strings.TrimPrefix(id, "0x")
}

func isKnownUSBSerial(vid, pid string) bool {
	vid = normalizeUSBID(vid)
	pid = normalizeUSBID(pid)

	switch vid {
	case "1a86": // QinHeng CH340/CH341/CH343
		return pid == "7523" || pid == "5523" || pid == "7522" || pid == "55d4"
	case "10c4": // Silicon Labs CP210x
		return pid == "ea60" || pid == "ea70"
	case "0403": // FTDI
		return pid == "6001" || pid == "6010" || pid == "6015"
	case "2341", "2a03": // Arduino LLC / Arduino.org
		return true
	case "1b4f": // SparkFun
		return true
	}

	return false
}

func describeUSBDevices(devices []usbSerialDevice) string {
	if len(devices) == 0 {
		return "none"
	}

	parts := make([]string, 0, len(devices))
	for _, device := range devices {
		description := fmt.Sprintf("%s (%s:%s", device.name, device.vid, device.pid)
		if device.product != "" {
			description += " " + device.product
		}
		description += ")"
		parts = append(parts, description)
	}

	return strings.Join(parts, ", ")
}
