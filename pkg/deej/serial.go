package deej

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/jacobsa/go-serial/serial"
	"github.com/omriharel/deej/pkg/deej/util"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

// SerialIO provides a deej-aware abstraction layer to managing serial I/O
type SerialIO struct {
	comPort  string
	baudRate uint

	deej   *Deej
	logger *zap.SugaredLogger

	stopChannel chan bool
	connected   bool
	connOptions serial.OpenOptions
	conn        io.ReadWriteCloser

	lastKnownNumSliders int
	currentVolumeDatas  []VolumeData

	sliderMoveConsumers []chan SliderEvent
}

type VolumeData struct {
	Value float32
	Mute  bool
}

type ArduinoData struct {
	Value      int
	ToggleMute bool
}

// SliderEvent represents a single slider move captured by deej
type SliderEvent struct {
	SliderID     int
	PercentValue float32
	ToggleMute   bool
}

var expectedLinePattern = regexp.MustCompile(`^-?\d{1,4}(\|-?\d{1,4})*\r\n$`)

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
		sliderMoveConsumers: []chan SliderEvent{},
	}

	logger.Debug("Created serial i/o instance")

	// respond to config changes
	sio.setupOnConfigReload()

	return sio, nil
}

// Start attempts to connect to our arduino chip
func (sio *SerialIO) Start() error {
	// don't allow multiple concurrent connections
	if sio.connected {
		sio.logger.Warn("Already connected, can't start another without closing first")
		return errors.New("serial: connection already active")
	}

	// set minimum read size according to platform (0 for windows, 1 for linux)
	// this prevents a rare bug on windows where serial reads get congested,
	// resulting in significant lag
	minimumReadSize := 0
	if util.Linux() {
		minimumReadSize = 1
	}

	sio.connOptions = serial.OpenOptions{
		PortName:        sio.deej.config.ConnectionInfo.COMPort,
		BaudRate:        uint(sio.deej.config.ConnectionInfo.BaudRate),
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: uint(minimumReadSize),
	}

	sio.logger.Debugw("Attempting serial connection",
		"comPort", sio.connOptions.PortName,
		"baudRate", sio.connOptions.BaudRate,
		"minReadSize", minimumReadSize)

	var err error
	sio.conn, err = serial.Open(sio.connOptions)
	if err != nil {

		// might need a user notification here, TBD
		sio.logger.Warnw("Failed to open serial connection", "error", err)
		return fmt.Errorf("open serial connection: %w", err)
	}

	namedLogger := sio.logger.Named(strings.ToLower(sio.connOptions.PortName))

	namedLogger.Infow("Connected", "conn", sio.conn)
	sio.connected = true

	// read lines or await a stop
	go func() {
		connReader := bufio.NewReader(sio.conn)
		bytesChannel := sio.readBytes(namedLogger, connReader)

		for {
			select {
			case <-sio.stopChannel:
				sio.close(namedLogger)
			case bytes := <-bytesChannel:
				sio.handleBytes(namedLogger, bytes)
			}
		}
	}()

	return nil
}

// Stop signals us to shut down our serial connection, if one is active
func (sio *SerialIO) Stop() {
	if sio.connected {
		sio.logger.Debug("Shutting down serial connection")
		sio.stopChannel <- true
	} else {
		sio.logger.Debug("Not currently connected, nothing to stop")
	}
}

// SubscribeToSliderMoveEvents returns an unbuffered channel that receives
// a sliderMoveEvent struct every time a slider moves
func (sio *SerialIO) SubscribeToSliderMoveEvents() chan SliderEvent {
	ch := make(chan SliderEvent)
	sio.sliderMoveConsumers = append(sio.sliderMoveConsumers, ch)

	return ch
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
					sio.Stop()

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

func (sio *SerialIO) readBytes(logger *zap.SugaredLogger, reader *bufio.Reader) chan []byte {
	ch := make(chan []byte)

	go func() {
		for {
			b, _ := reader.ReadByte()

			if b != 0xAA {
				continue
			}

			frameSize := len(sio.deej.config.SliderMapping.m)*2 + 1
			payload := make([]byte, frameSize)

			if _, err := io.ReadFull(reader, payload); err != nil {
				logger.Warnw("Failed to read bytes from serial", "error", err)
				close(ch)
				return
			}

			if sio.deej.Verbose() {
				logger.Debugw("got frame", "len", len(payload), "hex", fmt.Sprintf("% X", payload))
			}

			if payload[len(payload)-1] != 0x55 {
				logger.Debugw("wrong end byte", "byte", payload[frameSize-1])
				continue
			}

			ch <- payload[:frameSize-1]
		}
	}()

	return ch
}

func (sio *SerialIO) handleBytes(logger *zap.SugaredLogger, bytes []byte) {
	data := []ArduinoData{}

	if len(bytes) != len(sio.deej.config.SliderMapping.m)*2 {
		logger.Warnw("Wrong number of bytes received", "bytes number", len(bytes))
		return
	}

	for i := 0; i < len(bytes); i += 2 {
		data = append(data, ArduinoData{})
		newDataIdx := len(data) - 1

		packed := uint16(bytes[i])<<8 | uint16(bytes[i+1])

		data[newDataIdx].ToggleMute = (packed>>11)&0x01 != 0

		rawValue := packed & 0x07FF

		if rawValue&0x0400 != 0 {
			data[newDataIdx].Value = int(int16(rawValue | 0xF800))
		} else {
			data[newDataIdx].Value = int(rawValue)
		}
	}

	logger.Debugw("Reconstructed data", "data", data)

	numSliders := len(data)

	// update our slider count, if needed - this will send slider move events for all
	if numSliders != sio.lastKnownNumSliders {
		logger.Infow("Detected sliders", "amount", numSliders)
		sio.lastKnownNumSliders = numSliders
		sio.currentVolumeDatas = make([]VolumeData, numSliders)

		// reset everything to be an impossible value to force the slider move event later
		for idx := range sio.currentVolumeDatas {
			sio.currentVolumeDatas[idx].Value = -1.0
		}
	}

	// for each slider:
	sliderEvents := []SliderEvent{}
	for sliderIdx, arduinoData := range data {

		number := arduinoData.Value

		// turns out the first line could come out dirty sometimes (i.e. "4558|925|41|643|220")
		// so let's check the first number for correctness just in case
		if sliderIdx == 0 && number > 1023 {
			sio.logger.Debugw("Got malformed line from serial, ignoring", "data", arduinoData)
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

		additive := slices.Contains(sio.deej.config.AdditiveIndices, sliderIdx)

		if additive {
			finalVolume := sio.deej.sessions.getCurrentVolume(sliderIdx)

			if number != 0 {
				finalVolume += normalizedScalar

				if finalVolume < 0 {
					finalVolume = 0
				}
				if finalVolume > 1 {
					finalVolume = 1
				}
			}

			normalizedScalar = finalVolume
		}

		if sio.deej.config.UseLogVolume && !additive {
			normalizedScalar = LinearToLog(normalizedScalar)
			logger.Infow("Bent linear to Log", "Linear", dirtyFloat, "Log", normalizedScalar)
		}

		// check if it changes the desired state (could just be a jumpy raw slider value)
		// significantlyDifferent := util.SignificantlyDifferent(sio.currentVolumeDatas[sliderIdx].Value, normalizedScalar, sio.deej.config.NoiseReductionLevel)
		significantlyDifferent := true

		if significantlyDifferent || arduinoData.ToggleMute {

			// if it does, update the saved value and create a move event
			sio.currentVolumeDatas[sliderIdx].Value = normalizedScalar
			sio.currentVolumeDatas[sliderIdx].Mute = !sio.currentVolumeDatas[sliderIdx].Mute

			sliderEvents = append(sliderEvents, SliderEvent{
				SliderID:     sliderIdx,
				PercentValue: normalizedScalar,
				ToggleMute:   arduinoData.ToggleMute,
			})

			if sio.deej.Verbose() {
				logger.Debugw("Slider event", "event", sliderEvents[len(sliderEvents)-1])
			}
		}
	}

	// deliver move events if there are any, towards all potential consumers
	if len(sliderEvents) > 0 {
		for _, consumer := range sio.sliderMoveConsumers {
			for _, moveEvent := range sliderEvents {
				consumer <- moveEvent
			}
		}
	}
}

const (
	minDB = -60.0
	maxDB = 0.0
)

// FromLinear converts a 0.0–1.0 UI value into an amplitude multiplier.
func LinearToLog(x float32) float32 {
	// Clamp just in case.
	if x <= 0 {
		return 0
	} else if x >= 1 {
		return 1
	}

	// Map the linear position to decibels.
	db := minDB + x*(maxDB-minDB) // lerp

	// Convert dB to amplitude. (dB = 20 * log10(A))
	amp := math.Pow(10.0, float64(db/20.0))

	return float32(amp)
}
