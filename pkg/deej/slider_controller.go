package deej

type SliderController interface {
	Start() error
	Stop()
	SubscribeToSliderMoveEvents() chan SliderMoveEvent
}

// SliderMoveEvent represents a single slider move captured by deej
type SliderMoveEvent struct {
	SliderID     int
	PercentValue float32
}
