package deej

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jacobsa/go-serial/serial"
	"go.uber.org/zap"

	"github.com/omriharel/deej/util"
)

// SerialIO provides a deej-aware abstraction layer to managing serial I/O
type SerialIO struct {
	comPort  string
	baudRate uint

	deej        *Deej
	logger      *zap.SugaredLogger
	namedLogger *zap.SugaredLogger
	stopChannel chan bool
	connected   bool
	connOptions serial.OpenOptions
	conn        io.ReadWriteCloser

	lastKnownNumSliders        int
	currentSliderPercentValues []float32

	sliderMoveConsumers []chan SliderMoveEvent
}

// SliderMoveEvent represents a single slider move captured by deej
type SliderMoveEvent struct {
	SliderID     int
	PercentValue float32
}

var expectedLinePattern = regexp.MustCompile(`^\d{1,4}(\|\d{1,4})*\r\n$`)

// NewSerialIO creates a SerialIO instance that uses the provided deej
// instance's connection info to establish communications with the arduino chip
func NewSerialIO(deej *Deej, logger *zap.SugaredLogger) (*SerialIO, error) {
	logger = logger.Named("serial")

	sio := &SerialIO{
		deej:                deej,
		logger:              logger,
		stopChannel:         make(chan bool),
		connected:           false,
		conn:                nil,
		sliderMoveConsumers: []chan SliderMoveEvent{},
	}

	logger.Debug("Created serial i/o instance")

	// respond to config changes
	sio.setupOnConfigReload()

	return sio, nil
}

// Initialize Start the Serial Port
func (sio *SerialIO) Initialize() error {
	// don't allow multiple concurrent connections
	if sio.connected {
		sio.logger.Warn("Already connected, can't start another without closing first")
		return errors.New("serial: connection already active")
	}

	sio.connOptions = serial.OpenOptions{
		PortName:        sio.deej.config.ConnectionInfo.COMPort,
		BaudRate:        uint(sio.deej.config.ConnectionInfo.BaudRate),
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 1,
	}

	sio.namedLogger = sio.logger.Named(strings.ToLower(sio.connOptions.PortName))

	sio.logger.Debugw("Attempting serial connection",
		"comPort", sio.connOptions.PortName,
		"baudRate", sio.connOptions.BaudRate)

	var err error
	sio.conn, err = serial.Open(sio.connOptions)
	if err != nil {

		// might need a user notification here, TBD
		sio.namedLogger.Warnw("Failed to open serial connection", "error", err)
		return fmt.Errorf("open serial connection: %w", err)
	}

	sio.namedLogger.Infow("Connected", "conn", sio.conn)
	sio.connected = true

	return nil
}

// Start attempts to connect to our arduino chip
func (sio *SerialIO) Start() error {
	// read lines or await a stop

	go func() {

		lineChannel := sio.readLine(sio.namedLogger)

		for {

			select {
			case <-sio.stopChannel:
				lineChannel = nil
				sio.close(sio.namedLogger)
				return
			default:
				sio.WriteStringLine(sio.namedLogger, "deej.core.values")
				line := <-lineChannel
				sio.handleLine(sio.namedLogger, line)
			}
		}
	}()

	return nil
}

// Pause stops active polling for use resume with start
func (sio *SerialIO) Pause() {
	<-sio.stopChannel
}

// Shutdown signals us to shut down our serial connection, if one is active
func (sio *SerialIO) Shutdown() {
	if sio.connected {
		sio.logger.Debug("Shutting down serial connection")
		sio.stopChannel <- true
	} else {
		sio.logger.Debug("Not currently connected, nothing to stop")
	}
}

// SubscribeToSliderMoveEvents returns an unbuffered channel that receives
// a sliderMoveEvent struct every time a slider moves
func (sio *SerialIO) SubscribeToSliderMoveEvents() chan SliderMoveEvent {
	ch := make(chan SliderMoveEvent)
	sio.sliderMoveConsumers = append(sio.sliderMoveConsumers, ch)

	return ch
}

// WriteStringLine retruns nothing
// Writes a string to the serial port
func (sio *SerialIO) WriteStringLine(logger *zap.SugaredLogger, line string) {
	_, err := sio.conn.Write([]byte(line))
	if err != nil {

		// we probably don't need to log this, it'll happen once and the read loop will stop
		// logger.Warnw("Failed to read line from serial", "error", err, "line", line)
		return
	}
	_, err = sio.conn.Write([]byte("\n"))

	if err != nil {

		// we probably don't need to log this, it'll happen once and the read loop will stop
		// logger.Warnw("Failed to read line from serial", "error", err, "line", line)
		return
	}
}

// WaitFor returns nothing
// Waits for the specified line befor continueing
func (sio *SerialIO) WaitFor(logger *zap.SugaredLogger, cmdKey string) (success bool) {
	for {
		line := <-sio.readLine(logger)
		if len(line) > 1 {
			if line == cmdKey {
				return true
			}
			logger.Error("Serial Device Error: " + line)
			return false
		}
	}
}

// WriteBytesLine retruns nothing
// Writes a byteArray to the serial port
func (sio *SerialIO) WriteBytesLine(logger *zap.SugaredLogger, line []byte) {
	_, err := sio.conn.Write([]byte(line))
	if err != nil {

		// we probably don't need to log this, it'll happen once and the read loop will stop
		// logger.Warnw("Failed to read line from serial", "error", err, "line", line)
		return
	}
	_, err = sio.conn.Write([]byte("\n"))

	if err != nil {

		// we probably don't need to log this, it'll happen once and the read loop will stop
		// logger.Warnw("Failed to read line from serial", "error", err, "line", line)
		return
	}
}

func (sio *SerialIO) setupOnConfigReload() {
	configReloadedChannel := sio.deej.config.SubscribeToChanges()

	const stopDelay = 50 * time.Millisecond

	go func() {
		for {
			select {
			case <-configReloadedChannel:

				// make any config reload unset our slider number to ensure process volumes are being re-set
				// (the next read line will emit SliderMoveEvent instances for all sliders)\
				// this needs to happen after a small delay, because the session map will also re-acquire sessions
				// whenever the config file is reloaded, and we don't want it to receive these move events while the map
				// is still cleared. this is kind of ugly, but shouldn't cause any issues
				go func() {
					<-time.After(stopDelay)
					sio.lastKnownNumSliders = 0
				}()

				// if connection params have changed, attempt to stop and start the connection
				if sio.deej.config.ConnectionInfo.COMPort != sio.connOptions.PortName ||
					uint(sio.deej.config.ConnectionInfo.BaudRate) != sio.connOptions.BaudRate {

					sio.logger.Info("Detected change in connection parameters, attempting to renew connection")
					sio.Shutdown()

					// let the connection close
					<-time.After(stopDelay)

					if err := sio.Start(); err != nil {
						sio.logger.Warnw("Failed to renew connection after parameter change", "error", err)
					} else {
						sio.logger.Debug("Renewed connection successfully")
					}
				}
			}
		}
	}()
}

func (sio *SerialIO) close(logger *zap.SugaredLogger) {
	if err := sio.conn.Close(); err != nil {
		logger.Warnw("Failed to close serial connection", "error", err)
	} else {
		logger.Debug("Serial connection closed")
	}

	sio.conn = nil
	sio.connected = false
}

func (sio *SerialIO) readLine(logger *zap.SugaredLogger) chan string {
	ch := make(chan string)

	go func() {
		for {
			line, err := bufio.NewReader(sio.conn).ReadString('\n')
			if err != nil {

				// we probably don't need to log this, it'll happen once and the read loop will stop
				// logger.Warnw("Failed to read line from serial", "error", err, "line", line)
				return
			}

			// no reason to log here, just deliver the line to the channel
			// logger.Debugw("Read new line", "line", line)
			ch <- line
		}
	}()

	return ch
}

func (sio *SerialIO) handleLine(logger *zap.SugaredLogger, line string) {

	// this function receives an unsanitized line which is guaranteed to end with LF,
	// but most lines will end with CRLF. it may also have garbage instead of
	// deej-formatted values, so we must check for that! just ignore bad ones
	if !expectedLinePattern.MatchString(line) {
		return
	}

	// trim the suffix
	line = strings.TrimSuffix(line, "\r\n")

	// split on pipe (|), this gives a slice of numerical strings between "0" and "1023"
	splitLine := strings.Split(line, "|")
	numSliders := len(splitLine)

	// update our slider count, if needed - this will send slider move events for all
	if numSliders != sio.lastKnownNumSliders {
		logger.Infow("Detected sliders", "amount", numSliders)
		sio.lastKnownNumSliders = numSliders
		sio.currentSliderPercentValues = make([]float32, numSliders)

		// reset everything to be an impossible value to force the slider move event later
		for idx := range sio.currentSliderPercentValues {
			sio.currentSliderPercentValues[idx] = -1.0
		}
	}

	// for each slider:
	moveEvents := []SliderMoveEvent{}
	for sliderIdx, stringValue := range splitLine {

		// convert string values to integers ("1023" -> 1023)
		number, _ := strconv.Atoi(stringValue)

		// turns out the first line could come out dirty sometimes (i.e. "4558|925|41|643|220")
		// so let's check the first number for correctness just in case
		if sliderIdx == 0 && number > 1023 {
			sio.logger.Debugw("Got malformed line from serial, ignoring", "line", line)
			return
		}

		// map the value from raw to a "dirty" float between 0 and 1 (e.g. 0.15451...)
		dirtyFloat := float32(number) / 1023.0

		// normalize it to an actual volume scalar between 0.0 and 1.0 with 2 points of precision
		normalizedScalar := util.NormalizeScalar(dirtyFloat)

		// if sliders are inverted, take the complement of 1.0
		if sio.deej.config.InvertSliders {
			normalizedScalar = 1 - normalizedScalar
		}

		// check if it changes the desired state (could just be a jumpy raw slider value)
		if util.SignificantlyDifferent(sio.currentSliderPercentValues[sliderIdx], normalizedScalar) {

			// if it does, update the saved value and create a move event
			sio.currentSliderPercentValues[sliderIdx] = normalizedScalar

			moveEvents = append(moveEvents, SliderMoveEvent{
				SliderID:     sliderIdx,
				PercentValue: normalizedScalar,
			})

			// like in other places, this is too much to log - we'll log actual target volume events later
			// logger.Debugw("Slider moved", "event", moveEvents[len(moveEvents)-1])
		}
	}

	// deliver move events if there are any, towards all potential consumers
	if len(moveEvents) > 0 {
		for _, consumer := range sio.sliderMoveConsumers {
			for _, moveEvent := range moveEvents {
				consumer <- moveEvent
			}
		}
	}
}
