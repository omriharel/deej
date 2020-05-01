package deej

import (
	"github.com/getlantern/systray"

	"github.com/omriharel/deej/icon"
	"github.com/omriharel/deej/util"
)

func (d *Deej) initializeTray(onDone func()) {
	logger := d.logger.Named("tray")

	onReady := func() {
		logger.Debug("Tray instance ready")

		systray.SetTemplateIcon(icon.Data, icon.Data)
		systray.SetTitle("deej")
		systray.SetTooltip("deej")

		editConfig := systray.AddMenuItem("Edit configuration", "Open config file with notepad")
		quit := systray.AddMenuItem("Quit", "Stop deej and quit")

		// wait on things to happen
		go func() {
			for {
				select {

				// quit
				case <-quit.ClickedCh:
					logger.Debug("Quit menu item clicked, stopping")

					d.signalStop()

					logger.Debug("Quitting tray")
					systray.Quit()

				// edit config
				case <-editConfig.ClickedCh:
					logger.Debug("Edit config menu item clicked, opening config for editing")

					if err := util.OpenExternal(logger, "notepad.exe", configFilepath); err != nil {
						logger.Warnw("Failed to open config file for editing", "error", err)
					}
				}
			}
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
