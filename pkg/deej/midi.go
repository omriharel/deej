package deej

import (
	"github.com/omriharel/deej/pkg/deej/util"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	"time"

	//"errors"
	"fmt"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
	"io"
	//"regexp"
	"go.uber.org/zap"
	//"time"

	//"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

// MidiIO provides a deej-aware abstraction layer to managing midi I/O
type MidiIO struct {
	midi_indev    drivers.In
	midi_drv      drivers.Driver
	midi_stop     func()
	midi_dev_name string

	deej   *Deej
	logger *zap.SugaredLogger

	//stopChannel chan bool
	connected bool
	//connOptions midi.OpenOptions
	conn io.ReadWriteCloser

	lastKnownNumSliders        uint8
	currentSliderPercentValues []float32

	sliderMoveConsumers []chan SliderMoveEvent
}

// SliderMoveEvent represents a single slider move captured by deej
//type SliderMoveEvent struct {
//	SliderID     int
//	PercentValue float32
//}

//var expectedLinePattern = regexp.MustCompile(`^\d{1,4}(\|\d{1,4})*\r\n$`)

// NewMidiIO creates a MidiIO instance that uses the provided deej
// instance's connection info to establish communications with the arduino chip
func NewMidiIO(deej *Deej, logger *zap.SugaredLogger) (*MidiIO, error) {
	logger = logger.Named("midi")

	mio := &MidiIO{
		deej:   deej,
		logger: logger,
		//stopChannel:         make(chan bool),
		connected:           false,
		conn:                nil,
		sliderMoveConsumers: []chan SliderMoveEvent{},
		midi_dev_name:       deej.config.ConnectionInfo.MidiDevName,
	}
	fmt.Println("devname use:" + mio.midi_dev_name + "")

	logger.Debug("Created midi i/o instance")

	// respond to config changes
	mio.setupOnConfigReload()

	return mio, nil
}

// Start attempts to connect to our arduino chip
func (mio *MidiIO) Start() error {

	mio.midi_dev_name = mio.deej.config.ConnectionInfo.MidiDevName

	if mio.deej.config.ConnectionInfo.ConnectionType != "midi" {
		mio.logger.Debug("Not using midi connection")
		return nil
	}

	if mio.midi_dev_name == "" {
		mio.logger.Error("MIDI device name is nil or empty")
		return nil
	}

	// Initialize the MIDI driver
	drv, err := rtmididrv.New()
	if err != nil {
		fmt.Println("Failed to initialize MIDI driver:", err)
		return nil
	}
	mio.midi_drv = drv
	//defer drv.Close()

	// List available MIDI input devices
	ins, err := drv.Ins()
	if err != nil {
		fmt.Println("Failed to list input devices:", err)
		return nil
	}

	if len(ins) == 0 {
		fmt.Println("No MIDI input devices found.")
		return nil
	}

	fmt.Println("Available MIDI Input Devices:")
	for i, in := range ins {
		fmt.Printf("[%d] %s\n", i, in.String())
	}

	fmt.Printf("Using device: %s\n", mio.midi_dev_name)

	// Open the selected MIDI input device
	in, err := midi.FindInPort(mio.midi_dev_name)
	if in == nil || err != nil {
		fmt.Println("Failed to find MIDI input device:", err)
		return nil
	}

	if err := in.Open(); err != nil {
		fmt.Println("Failed to open MIDI input device:", err)
		return nil
	}
	mio.midi_indev = in

	stop, err := midi.ListenTo(in, func(msg midi.Message, timestampms int32) {
		var bt []byte
		var ch, key, vel uint8
		switch {
		case msg.GetSysEx(&bt):
		case msg.GetControlChange(&ch, &key, &vel):
			//fmt.Printf("Control Change - Channel: %d, Key: %d, Value: %d\n", ch, key, vel)
			mio.handleUpdate(mio.logger, ch, key, vel)
		default:
			fmt.Printf("got Unknown type: % X\n", bt)
		}
	}, midi.UseSysEx())

	mio.midi_stop = stop
	if err != nil {
		fmt.Println("Failed to listen to MIDI input device:", err)
	}

	//stop()

	return nil
}

// Stop signals us to shut down our midi connection, if one is active
func (mio *MidiIO) Stop() {
	if mio.connected {
		mio.logger.Debug("Shutting down midi connection")
		//mio.stopChannel <- true
		mio.midi_stop()
	} else {
		mio.logger.Debug("Not currently connected, nothing to stop")
	}
}

// SubscribeToSliderMoveEvents returns an unbuffered channel that receives
// a sliderMoveEvent struct every time a slider moves
func (mio *MidiIO) SubscribeToSliderMoveEvents() chan SliderMoveEvent {
	ch := make(chan SliderMoveEvent)
	mio.sliderMoveConsumers = append(mio.sliderMoveConsumers, ch)

	return ch
}

func (mio *MidiIO) setupOnConfigReload() {
	configReloadedChannel := mio.deej.config.SubscribeToChanges()

	const stopDelay = 50 * time.Millisecond

	go func() {
		for {
			select {
			case <-configReloadedChannel:

				fmt.Println("Detected config reload, attempting to re-acquire all audio sessions")

				// make any config reload unset our slider number to ensure process volumes are being re-set
				// (the next read line will emit SliderMoveEvent instances for all sliders)\
				// this needs to happen after a small delay, because the session map will also re-acquire sessions
				// whenever the config file is reloaded, and we don't want it to receive these move events while the map
				// is still cleared. this is kind of ugly, but shouldn't cause any issues
				go func() {
					<-time.After(stopDelay)
					mio.lastKnownNumSliders = 0
				}()

				// if connection params have changed, attempt to stop and start the connection
				if mio.deej.config.ConnectionInfo.MidiDevName != mio.midi_dev_name || mio.deej.config.ConnectionInfo.ConnectionType != "midi" {

					mio.logger.Info("Detected change in connection parameters, attempting to renew connection")
					mio.Stop()

					// let the connection close
					<-time.After(stopDelay)

					if err := mio.Start(); err != nil {
						mio.logger.Warnw("Failed to renew connection after parameter change", "error", err)
					} else {
						mio.logger.Debug("Renewed connection successfully")
					}
				}
			}
		}
	}()
}

func (mio *MidiIO) close(logger *zap.SugaredLogger) {
	if err := mio.conn.Close(); err != nil {
		logger.Warnw("Failed to close midi connection", "error", err)
	} else {
		logger.Debug("midi connection closed")
	}

	mio.conn = nil
	mio.connected = false
}

func (mio *MidiIO) handleUpdate(logger *zap.SugaredLogger, ch uint8, key uint8, vel uint8) {

	//Check if all sliders are actually registered!
	var numSliders uint8 = ch + 1 //Because channel count starts at 0
	if numSliders > mio.lastKnownNumSliders {
		logger.Infow("Detected sliders", "amount", ch)
		mio.lastKnownNumSliders = numSliders
		mio.currentSliderPercentValues = make([]float32, numSliders)

		// reset everything to be an impossible value to force the slider move event later
		for idx := range mio.currentSliderPercentValues {
			mio.currentSliderPercentValues[idx] = -1.0
		}
	}

	moveEvent := SliderMoveEvent{}
	moveEvent.SliderID = int(ch)
	moveEvent.PercentValue = float32(vel) / 127.0

	// if sliders are inverted, take the complement of 1.0
	if mio.deej.config.InvertSliders {
		moveEvent.PercentValue = 1.0 - moveEvent.PercentValue
	}

	// check if it changes the desired state (could just be a jumpy raw slider value)
	if util.SignificantlyDifferent(mio.currentSliderPercentValues[ch], moveEvent.PercentValue, mio.deej.config.NoiseReductionLevel) {

		// if it does, update the saved value and create a move event
		mio.currentSliderPercentValues[ch] = moveEvent.PercentValue

		if mio.deej.Verbose() {
			logger.Debugw("Slider moved", "event", moveEvent)
		}
	}

	for _, consumer := range mio.sliderMoveConsumers {
		consumer <- moveEvent
	}

}
