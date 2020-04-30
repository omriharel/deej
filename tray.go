package deej

import (
	"github.com/getlantern/systray"

	"github.com/omriharel/deej/icon"
)

func (d *Deej) initializeTray(onDone func()) {
	onReady := func() {
		systray.SetTemplateIcon(icon.Data, icon.Data)
		systray.SetTitle("deej")
		systray.SetTooltip("deej")

		quit := systray.AddMenuItem("Quit", "Stop deej and quit")

		// wait on menu quit
		go func() {
			<-quit.ClickedCh
			d.signalStop()
			systray.Quit()
		}()

		// actually start the main runtime
		onDone()
	}

	onExit := func() {}

	// start the tray icon
	systray.Run(onReady, onExit)
}
