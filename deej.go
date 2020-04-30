// Package deej provides a machine-side client that pairs with an Arduino
// chip to form a tactile, physical volume control system/
package deej

import (
	"fmt"
	"log"
	"os"

	"github.com/omriharel/deej/util"
)

const (

	// when this is set to anything, deej won't use a tray icon
	envNoTray = "DEEJ_NO_TRAY_ICON"
)

// Deej is the main entity managing access to all sub-components
type Deej struct {
	notifier Notifier

	stopChannel chan bool
}

// NewDeej creates a Deej instance
func NewDeej() (*Deej, error) {
	notifier, err := NewToastNotifier()
	if err != nil {
		return nil, fmt.Errorf("create new ToastNotifier: %w", err)
	}

	d := &Deej{
		notifier:    notifier,
		stopChannel: make(chan bool),
	}

	return d, nil
}

// Initialize sets up components and starts to run in the background
func (d *Deej) Initialize() error {
	if _, noTraySet := os.LookupEnv(envNoTray); noTraySet {

		// run in main thread while waiting on ctrl+C
		interruptChannel := util.SetupCloseHandler()

		go func() {
			<-interruptChannel
			d.signalStop()
		}()

		d.run()

	} else {
		d.initializeTray(d.run)
	}

	return nil
}

func (d *Deej) run() {

	// test notification (will be removed)
	err := d.notifier.Notify("Hello", "deej is running!")
	if err != nil {
		log.Fatalf("test notification: %v", err)
	}

	// wait until stopped
	<-d.stopChannel
	log.Printf("Stop signal received, terminating")
}

func (d *Deej) signalStop() {
	log.Printf("Signalling stop")
	d.stopChannel <- true
}
