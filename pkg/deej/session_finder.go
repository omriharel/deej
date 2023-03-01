package deej

// SessionFinder represents an entity that can find all current audio sessions
type SessionFinder interface {
	GetAllSessions() ([]Session, error)
	GetLevelMeterChannel() chan string
	Release() error
	GetSessionReloadEvent() chan bool
}
