package deej

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

type sessionMap struct {
	deej   *Deej
	logger *zap.SugaredLogger

	m    map[string][]Session
	lock sync.Locker
}

func newSessionMap(deej *Deej, logger *zap.SugaredLogger) (*sessionMap, error) {
	logger = logger.Named("sessions")

	m := &sessionMap{
		deej:   deej,
		logger: logger,
		m:      make(map[string][]Session),
		lock:   &sync.Mutex{},
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
	m.logger.Debug("me!")
}

func (m *sessionMap) get(key string) ([]Session, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	value, ok := m.m[key]
	return value, ok
}

func (m *sessionMap) set(key string, value []Session) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.m[key] = value
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
