package deej

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-ole/go-ole"
	"go.uber.org/zap"
)

type sessionMap struct {
	deej   *Deej
	logger *zap.SugaredLogger

	m    map[string][]Session
	lock sync.Locker

	eventCtx *ole.GUID // needed for some session actions to successfully notify other audio consumers
}

const (
	masterSessionName = "master" // master device volume
	systemSessionName = "system" // system sounds volume

	myteriousGUID = "{1ec920a1-7db8-44ba-9779-e5d28ed9f330}"
)

func newSessionMap(deej *Deej, logger *zap.SugaredLogger) (*sessionMap, error) {
	logger = logger.Named("sessions")

	m := &sessionMap{
		deej:     deej,
		logger:   logger,
		m:        make(map[string][]Session),
		lock:     &sync.Mutex{},
		eventCtx: ole.NewGUID(myteriousGUID),
	}

	logger.Debug("Created session map instance")

	return m, nil
}

func (m *sessionMap) initialize() error {
	if err := m.getAllSessions(); err != nil {
		m.logger.Warnw("Failed to get all sessions during session map initialization", "error", err)
		return fmt.Errorf("get all sessions during init: %w", err)
	}

	m.setupOnConfigReload()
	m.setupOnSliderMove()

	return nil
}

func (m *sessionMap) setupOnConfigReload() {
	configReloadedChannel := m.deej.config.SubscribeToChanges()

	go func() {
		for {
			select {
			case <-configReloadedChannel:
				m.logger.Debug("Detected config reload, attempting to re-acquire all audio sessions")

				// clear and release sessions first
				m.clear()

				if err := m.getAllSessions(); err != nil {
					m.logger.Warnw("Failed to re-acquire all audio sessions", "error", err)
				} else {
					m.logger.Debug("Re-acquired sessions successfully")
				}
			}
		}
	}()
}

func (m *sessionMap) setupOnSliderMove() {
	sliderEventsChannel := m.deej.serial.SubscribeToSliderMoveEvents()

	go func() {
		for {
			select {
			case event := <-sliderEventsChannel:
				m.handleSliderMoveEvent(event)
			}
		}
	}()
}

func (m *sessionMap) handleSliderMoveEvent(event SliderMoveEvent) {
	targets, ok := m.deej.config.SliderMapping.get(event.SliderID)

	// slider not found in config - silently ignore
	if !ok {
		return
	}

	targetFound := false

	// for each possible target for this slider...
	for _, target := range targets {

		// normalize the target name to match session keys
		normalizedTargetName := strings.ToLower(target)

		// check the map for matching sessions
		sessions, ok := m.get(normalizedTargetName)

		// no sessions matching this target - move on
		if !ok {
			continue
		}

		targetFound = true

		for _, session := range sessions {
			if session.GetVolume() != event.PercentValue {
				if err := session.SetVolume(event.PercentValue); err != nil {
					m.logger.Warnw("Failed to set target session volume", "error", err)
				}
			}
		}
	}

	if !targetFound {
		// ... consider refreshing sessions here on a cooldown
	}
}

func (m *sessionMap) add(value Session) {
	m.lock.Lock()
	defer m.lock.Unlock()

	key := value.Key()

	existing, ok := m.m[key]
	if !ok {
		m.m[key] = []Session{value}
	} else {
		existing = append(existing, value)
	}
}

func (m *sessionMap) get(key string) ([]Session, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	value, ok := m.m[key]
	return value, ok
}

func (m *sessionMap) clear() {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.logger.Debug("Releasing and clearing all audio sessions")

	for key, sessions := range m.m {
		for _, session := range sessions {
			session.Release()
		}

		delete(m.m, key)
	}

	m.logger.Debug("Session map cleared")
}

func (m *sessionMap) String() string {
	m.lock.Lock()
	defer m.lock.Unlock()

	sessionCount := 0

	for _, value := range m.m {
		sessionCount += len(value)
	}

	return fmt.Sprintf("<%d audio sessions>", sessionCount)
}
