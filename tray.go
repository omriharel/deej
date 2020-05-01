package deej

import (
	"github.com/getlantern/systray"

	"github.com/omriharel/deej/icon"
)

func (d *Deej) initializeTray(onDone func()) {
	logger := d.logger.Named("tray")

	onReady := func() {
		logger.Debug("Tray instance ready")

		systray.SetTemplateIcon(icon.Data, icon.Data)
		systray.SetTitle("deej")
		systray.SetTooltip("deej")

		quit := systray.AddMenuItem("Quit", "Stop deej and quit")

		// wait on menu quit
		go func() {
			<-quit.ClickedCh
			logger.Debug("Quit menu item clicked, stopping")

			d.signalStop()

			logger.Debug("Quitting tray")
			systray.Quit()
		}()

		// actually start the main runtime
		onDone()
	}

	onExit := func() {
		logger.Debug("Tray onExit called")
	}

	// start the tray icon
	logger.Debug("Running in tray")
	systray.Run(onReady, onExit)
}
