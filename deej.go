// Package deej provides a machine-side client that pairs with an Arduino
// chip to form a tactile, physical volume control system/
package deej

import (
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/omriharel/deej/util"
)

const (

	// when this is set to anything, deej won't use a tray icon
	envNoTray = "DEEJ_NO_TRAY_ICON"
)

// Deej is the main entity managing access to all sub-components
type Deej struct {
	logger   *zap.SugaredLogger
	notifier Notifier

	stopChannel chan bool
}

// NewDeej creates a Deej instance
func NewDeej(logger *zap.SugaredLogger) (*Deej, error) {
	logger = logger.Named("deej")

	notifier, err := NewToastNotifier(logger)
	if err != nil {
		logger.Errorw("Failed to create ToastNotifier", "error", err)
		return nil, fmt.Errorf("create new ToastNotifier: %w", err)
	}

	d := &Deej{
		logger:      logger,
		notifier:    notifier,
		stopChannel: make(chan bool),
	}

	logger.Debug("Created deej instance")

	return d, nil
}

// Initialize sets up components and starts to run in the background
func (d *Deej) Initialize() error {
	d.logger.Debug("Initializing")

	if _, noTraySet := os.LookupEnv(envNoTray); noTraySet {

		d.logger.Debugw("Running without tray icon", "reason", "envvar set")

		// run in main thread while waiting on ctrl+C
		interruptChannel := util.SetupCloseHandler()

		go func() {
			<-interruptChannel
			d.logger.Warn("Interrupted")
			d.signalStop()
		}()

		d.run()

	} else {
		d.initializeTray(d.run)
	}

	return nil
}

func (d *Deej) run() {
	d.logger.Info("Run loop starting")

	// wait until stopped
	<-d.stopChannel
	d.logger.Info("Stop channel signaled, terminating")
}

func (d *Deej) signalStop() {
	d.logger.Debug("Signalling stop channel")
	d.stopChannel <- true
}
