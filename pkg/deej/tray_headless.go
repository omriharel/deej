//go:build headless
// +build headless

package deej

// initializeTray is a no-op when deej is built with the headless tag. We still
// invoke onDone so the normal run loop starts, which makes headless builds work
// without requiring DEEJ_NO_TRAY_ICON.
func (d *Deej) initializeTray(onDone func()) {
	d.logger.Debug("Running without tray icon (headless build)")
	onDone()
}

func (d *Deej) stopTray() {
	d.logger.Debug("Headless build has no tray to stop")
}
