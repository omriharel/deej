package deej

import (
	"github.com/getlantern/systray"

	"github.com/jax-b/deej/icon"
	"github.com/jax-b/deej/util"
)

var (
	externalTrayText    []string
	externalTrayTooltip []string
	externalTrayItems   []chan *systray.MenuItem
	externalItems       bool = false
)

func (d *Deej) initializeTray(onDone func()) {
	logger := d.logger.Named("tray")

	onReady := func() {
		logger.Debug("Tray instance ready")

		systray.SetTemplateIcon(icon.DeejLogo, icon.DeejLogo)
		systray.SetTitle("deej")
		systray.SetTooltip("deej")

		if externalItems {
			for i, v := range externalTrayItems {
				newitem := systray.AddMenuItem(externalTrayText[i], externalTrayTooltip[i])
				v <- newitem
			}
			systray.AddSeparator()
		}

		editConfig := systray.AddMenuItem("Edit configuration", "Open config file with notepad")
		editConfig.SetIcon(icon.EditConfig)

		refreshSessions := systray.AddMenuItem("Re-scan audio sessions", "Manually refresh audio sessions if something's stuck")
		refreshSessions.SetIcon(icon.RefreshSessions)

		if d.version != "" {
			systray.AddSeparator()
			versionInfo := systray.AddMenuItem(d.version, "")
			versionInfo.Disable()
		}

		systray.AddSeparator()
		quit := systray.AddMenuItem("Quit", "Stop deej and quit")

		// wait on things to happen
		go func() {

			for {
				select {

				// quit
				case <-quit.ClickedCh:
					logger.Info("Quit menu item clicked, stopping")

					d.signalStop()

				// edit config
				case <-editConfig.ClickedCh:
					logger.Info("Edit config menu item clicked, opening config for editing")

					editor := "notepad.exe"
					if util.Linux() {
						editor = "gedit"
					}

					if err := util.OpenExternal(logger, editor, configFilepath); err != nil {
						logger.Warnw("Failed to open config file for editing", "error", err)
					}

				// refresh sessions
				case <-refreshSessions.ClickedCh:
					logger.Info("Refresh sessions menu item clicked, triggering session map refresh")
					d.sessions.refreshSessions(true)
				}
			}
		}()

		// for _, consumer := range externalTraySubscribers {
		// 	consumer <- true
		// }

		// actually start the main runtime
		onDone()
	}

	onExit := func() {
		logger.Debug("Tray exited")
	}

	// start the tray icon
	logger.Debug("Running in tray")
	systray.Run(onReady, onExit)

}

// AddMenuItem addes a menu item to the tray
func (d *Deej) AddMenuItem(text string, tooltip string) chan *systray.MenuItem {
	externalTrayText = append(externalTrayText, text)
	externalTrayTooltip = append(externalTrayTooltip, text)

	externalItems = true

	c := make(chan *systray.MenuItem)
	externalTrayItems = append(externalTrayItems, c)

	return c
}

func (d *Deej) stopTray() {
	d.logger.Debug("Quitting tray")
	systray.Quit()
}
