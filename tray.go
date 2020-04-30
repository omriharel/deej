package deej

import (
	"github.com/getlantern/systray"

	"github.com/omriharel/deej/icon"
)

func (d *Deej) initializeTray() {
	onReady := func() {
		systray.SetTemplateIcon(icon.Data, icon.Data)
		systray.SetTitle("deej")
		systray.SetTooltip("deej")

		quit := systray.AddMenuItem("Quit", "Stop deej and quit")

		go func() {
			<-quit.ClickedCh
			systray.Quit()
		}()
	}

	onExit := func() {}

	systray.Run(onReady, onExit)
}
