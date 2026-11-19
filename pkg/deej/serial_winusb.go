package deej

import (
	"bufio"
	"errors"
	"strings"
	"time"

	"github.com/jacobsa/go-serial/serial"
	"go.uber.org/zap"

	"github.com/omriharel/deej/pkg/deej/ch340"
)

// WinUSB mode: set `com_port: winusb` in config.yaml. deej then talks to the
// CH340 through Microsoft's WinUSB driver instead of a COM port, so the
// vendor CH341S64.SYS driver (which bluescreens) is never loaded.
//
// Unlike the COM port path, this mode keeps retrying: it waits for the board
// to be plugged in, and reconnects by itself if it is unplugged.

const (
	winusbPortName      = "winusb"
	winusbRetryInterval = 3 * time.Second
)

func isWinUSBPort(port string) bool {
	return strings.EqualFold(strings.TrimSpace(port), winusbPortName)
}

func (sio *SerialIO) startWinUSB() error {
	sio.connOptions = serial.OpenOptions{
		PortName: sio.deej.config.ConnectionInfo.COMPort,
		BaudRate: uint(sio.deej.config.ConnectionInfo.BaudRate),
	}

	sio.usbStop = make(chan struct{})
	sio.usbDone = make(chan struct{})
	sio.connected = true // the supervisor is running; Stop() must shut it down

	go sio.winusbSupervisor(sio.usbStop, sio.usbDone, int(sio.connOptions.BaudRate))
	return nil
}

func (sio *SerialIO) stopWinUSB() {
	close(sio.usbStop)
	<-sio.usbDone
	sio.usbStop = nil
	sio.usbDone = nil
	sio.conn = nil
	sio.connected = false
	sio.logger.Debug("WinUSB connection closed")
}

func (sio *SerialIO) winusbSupervisor(stop <-chan struct{}, done chan<- struct{}, baud int) {
	defer close(done)

	logger := sio.logger.Named(winusbPortName)
	lastErr := ""
	notified := false

	wait := func() bool {
		select {
		case <-stop:
			return false
		case <-time.After(winusbRetryInterval):
			return true
		}
	}

	for {
		dev, err := ch340.Open(baud)
		if err != nil {
			if err.Error() != lastErr {
				logger.Warnw("Can't open the CH340 via WinUSB, will keep retrying", "error", err)
				lastErr = err.Error()
			}
			if !notified && errors.Is(err, ch340.ErrWrongDriver) {
				sio.deej.notifier.Notify("deej can't use the controller",
					"The CH340 is still on its vendor driver. Switch it to WinUSB (see docs/winusb.md), or set com_port back to your COM port.")
				notified = true
			}
			if !wait() {
				return
			}
			continue
		}

		lastErr = ""
		notified = false
		sio.conn = dev
		logger.Infow("Connected", "conn", dev.String(), "baudRate", baud)

		lost := sio.pumpWinUSB(logger, dev, stop)
		dev.Close()
		sio.conn = nil
		sio.lastKnownNumSliders = 0 // re-send every slider value after reconnecting

		if !lost {
			return // stop requested
		}
		if !wait() {
			return
		}
	}
}

// pumpWinUSB feeds lines into handleLine until the device fails (returns true)
// or a stop is requested (returns false).
func (sio *SerialIO) pumpWinUSB(logger *zap.SugaredLogger, dev *ch340.Device, stop <-chan struct{}) bool {
	lines := make(chan string)
	readErr := make(chan error, 1)
	quit := make(chan struct{})
	defer close(quit)

	go func() {
		reader := bufio.NewReader(dev)

		// We join the board's stream mid-line, so the first line is usually a
		// fragment like "3" that would read as one slider at 0%. Drop it.
		if _, err := reader.ReadString('\n'); err != nil {
			readErr <- err
			return
		}

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				readErr <- err
				return
			}
			select {
			case lines <- line:
			case <-quit:
				return
			}
		}
	}()

	for {
		select {
		case <-stop:
			dev.Close() // unblocks the reader goroutine
			return false
		case line := <-lines:
			if sio.deej.Verbose() {
				logger.Debugw("Read new line", "line", line)
			}
			sio.handleLine(logger, line)
		case err := <-readErr:
			logger.Warnw("Lost the CH340, will reconnect when it's back", "error", err)
			return true
		}
	}
}
