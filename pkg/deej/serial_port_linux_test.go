package deej

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveSerialPortExplicitPath(t *testing.T) {
	port, err := resolveSerialPort("/dev/ttyUSB7", "", "")
	if err != nil {
		t.Fatalf("resolve explicit port: %v", err)
	}
	if port != "/dev/ttyUSB7" {
		t.Fatalf("expected explicit port, got %q", port)
	}
}

func TestResolveSerialPortAuto(t *testing.T) {
	port, err := resolveSerialPort(autoCOMPort, "", "")
	if err != nil {
		if strings.Contains(err.Error(), "no USB serial devices") ||
			strings.Contains(err.Error(), "no matching USB serial device") ||
			strings.Contains(err.Error(), "multiple USB serial devices match") {
			t.Skipf("auto-detection is not unambiguous here: %v", err)
		}
		t.Fatalf("resolve auto port: %v", err)
	}

	if !strings.HasPrefix(port, "/dev/") {
		t.Fatalf("expected a /dev path, got %q", port)
	}

	t.Logf("auto-detected serial port: %s", port)
}

func TestKnownUSBSerialIDs(t *testing.T) {
	cases := []struct {
		vid, pid string
		want     bool
	}{
		{"1a86", "7523", true},
		{"1A86", "7523", true},
		{"0x1a86", "0x7523", true},
		{"10c4", "ea60", true},
		{"2341", "0043", true},
		{"dead", "beef", false},
	}

	for _, tc := range cases {
		if got := isKnownUSBSerial(tc.vid, tc.pid); got != tc.want {
			t.Errorf("isKnownUSBSerial(%q, %q) = %v, want %v", tc.vid, tc.pid, got, tc.want)
		}
	}
}

func TestUSBSerialDeviceNames(t *testing.T) {
	cases := map[string]bool{
		"ttyUSB0":      true,
		"ttyACM0":      true,
		"ttyCH343USB0": true,
		"ttyS0":        false,
		"tty0":         false,
	}

	for name, want := range cases {
		if got := looksLikeUSBSerial(name); got != want {
			t.Errorf("looksLikeUSBSerial(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestUSBDeviceDirWalksUpToDevice(t *testing.T) {
	root, err := ioutil.TempDir("", "deej-usb-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	deviceDir := filepath.Join(root, "3-1")
	ifaceDir := filepath.Join(deviceDir, "3-1:1.0")
	ttyDir := filepath.Join(ifaceDir, "ttyUSB0")
	if err := os.MkdirAll(ttyDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(deviceDir, "idVendor"), []byte("1a86\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(deviceDir, "idProduct"), []byte("7523\n"), 0644); err != nil {
		t.Fatal(err)
	}

	deviceLink := filepath.Join(root, "ttyUSB0-device")
	if err := os.Symlink(ttyDir, deviceLink); err != nil {
		t.Fatal(err)
	}

	if got := usbDeviceDir(deviceLink); got != deviceDir {
		t.Fatalf("usbDeviceDir() = %q, want %q", got, deviceDir)
	}
}
