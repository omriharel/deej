package deej

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jacobsa/go-serial/serial"
	"go.uber.org/zap"

	"github.com/omriharel/deej/pkg/deej/util"
)

// SerialIO provides a deej-aware abstraction layer to managing serial I/O
type SerialIO struct {
	deej   *Deej
	logger *zap.SugaredLogger

	mu       sync.Mutex
	conn     io.ReadWriteCloser
	running  bool
	stopping bool
	wake     chan struct{}

	// remembered so config reloads only reconnect when connection settings
	// actually changed
	currentConfigPort string
	currentBaudRate   int

	stateMu                    sync.Mutex
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
		wake:                make(chan struct{}),
		sliderMoveConsumers: []chan SliderMoveEvent{},
	}

	logger.Debug("Created serial i/o instance")

	// respond to config changes
	sio.setupOnConfigReload()

	return sio, nil
}

// Start launches the serial supervisor. It keeps trying to discover and open
// the board, and reconnects automatically if the device disappears.
func (sio *SerialIO) Start() error {
	sio.mu.Lock()
	if sio.running {
		sio.mu.Unlock()
		return nil
	}

	if sio.stopping {
		sio.stopping = false
		sio.wake = make(chan struct{})
	}

	sio.running = true
	sio.mu.Unlock()

	go sio.run()

	return nil
}

func (sio *SerialIO) run() {
	defer func() {
		sio.mu.Lock()
		sio.running = false
		sio.mu.Unlock()
	}()

	backoff := time.Second
	const maxBackoff = 10 * time.Second

	for {
		if sio.isStopping() {
			return
		}

		connectionInfo := sio.deej.config.ConnectionInfo
		port, err := resolveSerialPort(connectionInfo.COMPort, connectionInfo.USBVID, connectionInfo.USBPID)
		if err != nil {
			sio.logger.Warnw("Failed to resolve serial port", "error", err)
			if !sio.wait(backoff) {
				return
			}
			backoff = nextBackoff(backoff, maxBackoff)
			continue
		}

		// set minimum read size according to platform (0 for windows, 1 for linux)
		// this prevents a rare bug on windows where serial reads get congested,
		// resulting in significant lag
		minimumReadSize := 0
		if util.Linux() {
			minimumReadSize = 1
		}

		connOptions := serial.OpenOptions{
			PortName:        port,
			BaudRate:        uint(connectionInfo.BaudRate),
			DataBits:        8,
			StopBits:        1,
			MinimumReadSize: uint(minimumReadSize),
		}

		sio.logger.Debugw("Attempting serial connection",
			"comPort", connOptions.PortName,
			"baudRate", connOptions.BaudRate,
			"minReadSize", minimumReadSize)

		conn, err := serial.Open(connOptions)
		if err != nil {
			sio.logger.Warnw("Failed to open serial connection", "comPort", port, "error", err)
			if !sio.wait(backoff) {
				return
			}
			backoff = nextBackoff(backoff, maxBackoff)
			continue
		}

		if sio.isStopping() {
			conn.Close()
			return
		}

		sio.setConnected(conn, connectionInfo.COMPort, connectionInfo.BaudRate)
		sio.logger.Infow("Connected", "comPort", port)
		backoff = time.Second

		sio.readLoop(port, conn)

		sio.clearConnected()
		if sio.isStopping() {
			return
		}

		sio.logger.Warnw("Serial connection lost, will retry", "comPort", port)
		if !sio.wait(backoff) {
			return
		}
		backoff = nextBackoff(backoff, maxBackoff)
	}
}

func (sio *SerialIO) readLoop(port string, conn io.ReadWriteCloser) {
	logger := sio.logger.Named(strings.ToLower(port))
	connReader := bufio.NewReader(conn)

	for {
		line, err := connReader.ReadString('\n')
		if err != nil {
			if sio.deej.Verbose() {
				logger.Warnw("Failed to read line from serial", "error", err, "line", line)
			}
			return
		}

		if sio.deej.Verbose() {
			logger.Debugw("Read new line", "line", line)
		}

		sio.handleLine(logger, line)
	}
}

func (sio *SerialIO) setConnected(conn io.ReadWriteCloser, configPort string, baudRate int) {
	sio.mu.Lock()
	sio.conn = conn
	sio.currentConfigPort = configPort
	sio.currentBaudRate = baudRate
	sio.mu.Unlock()

	sio.resetSliderState()
}

func (sio *SerialIO) clearConnected() {
	sio.mu.Lock()
	conn := sio.conn
	sio.conn = nil
	sio.mu.Unlock()

	if conn != nil {
		conn.Close()
	}
}

func (sio *SerialIO) isStopping() bool {
	sio.mu.Lock()
	defer sio.mu.Unlock()

	return sio.stopping
}

func (sio *SerialIO) wait(d time.Duration) bool {
	sio.mu.Lock()
	wake := sio.wake
	stopping := sio.stopping
	sio.mu.Unlock()

	if stopping || wake == nil {
		return false
	}

	select {
	case <-time.After(d):
		return true
	case <-wake:
		return false
	}
}

func nextBackoff(current, max time.Duration) time.Duration {
	current *= 2
	if current > max {
		current = max
	}

	return current
}

// Stop signals us to shut down our serial connection, if one is active
func (sio *SerialIO) Stop() {
	sio.mu.Lock()
	defer sio.mu.Unlock()

	if sio.stopping {
		return
	}

	sio.logger.Debug("Shutting down serial connection")
	sio.stopping = true

	if sio.conn != nil {
		sio.conn.Close()
		sio.conn = nil
	}

	if sio.wake != nil {
		close(sio.wake)
		sio.wake = nil
	}
}

// Reload closes the current connection so the supervisor re-discovers and
// reopens the board. It is used when connection settings change.
func (sio *SerialIO) Reload() {
	sio.mu.Lock()
	conn := sio.conn
	sio.conn = nil
	sio.mu.Unlock()

	if conn != nil {
		conn.Close()
	}
}

// SubscribeToSliderMoveEvents returns an unbuffered channel that receives
// a sliderMoveEvent struct every time a slider moves
func (sio *SerialIO) SubscribeToSliderMoveEvents() chan SliderMoveEvent {
	ch := make(chan SliderMoveEvent)
	sio.sliderMoveConsumers = append(sio.sliderMoveConsumers, ch)

	return ch
}

func (sio *SerialIO) setupOnConfigReload() {
	configReloadedChannel := sio.deej.config.SubscribeToChanges()

	const stopDelay = 50 * time.Millisecond

	go func() {
		for range configReloadedChannel {
			// Make any config reload unset our slider number to ensure process
			// volumes are re-set: the next read line will emit events for all
			// sliders. Delay this until the session map has re-acquired sessions.
			go func() {
				<-time.After(stopDelay)
				sio.resetSliderState()
			}()

			if sio.connectionSettingsChanged() {
				sio.logger.Info("Detected change in connection parameters, reconnecting")
				sio.Reload()
			}
		}
	}()
}

func (sio *SerialIO) connectionSettingsChanged() bool {
	sio.mu.Lock()
	defer sio.mu.Unlock()

	connectionInfo := sio.deej.config.ConnectionInfo

	return connectionInfo.COMPort != sio.currentConfigPort ||
		connectionInfo.BaudRate != sio.currentBaudRate
}

func (sio *SerialIO) resetSliderState() {
	sio.stateMu.Lock()
	defer sio.stateMu.Unlock()

	sio.lastKnownNumSliders = 0
	sio.currentSliderPercentValues = nil
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

	// A corrupted line with a missing value would shift every following
	// slider onto the wrong target. Ignore lines that don't contain at least
	// every slider referenced by the config.
	expectedSliders := sio.deej.config.SliderMapping.length()
	if expectedSliders > 0 && numSliders < expectedSliders {
		sio.logger.Debugw("Ignoring malformed line with too few sliders",
			"line", line,
			"sliders", numSliders,
			"expectedAtLeast", expectedSliders)
		return
	}

	sio.stateMu.Lock()

	// update our slider count, if needed - this will send slider move events for all
	if numSliders != sio.lastKnownNumSliders || len(sio.currentSliderPercentValues) != numSliders {
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
			sio.stateMu.Unlock()
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
		if util.SignificantlyDifferent(sio.currentSliderPercentValues[sliderIdx], normalizedScalar, sio.deej.config.NoiseReductionLevel) {
			// if it does, update the saved value and create a move event
			sio.currentSliderPercentValues[sliderIdx] = normalizedScalar

			moveEvents = append(moveEvents, SliderMoveEvent{
				SliderID:     sliderIdx,
				PercentValue: normalizedScalar,
			})

			if sio.deej.Verbose() {
				logger.Debugw("Slider moved", "event", moveEvents[len(moveEvents)-1])
			}
		}
	}

	sio.stateMu.Unlock()

	// deliver move events if there are any, towards all potential consumers
	if len(moveEvents) > 0 {
		for _, consumer := range sio.sliderMoveConsumers {
			for _, moveEvent := range moveEvents {
				consumer <- moveEvent
			}
		}
	}
}
