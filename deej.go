package deej

type Deej struct {
}

func (d *Deej) Initialize() error {
	d.initializeTray()
	return nil
}

func (d *Deej) Run() {

}
