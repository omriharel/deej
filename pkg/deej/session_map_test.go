package deej

import (
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

type fakeSession struct {
	baseSession

	volume   float32
	stale    bool
	setCalls int
}

func (s *fakeSession) GetVolume() float32 {
	return s.volume
}

func (s *fakeSession) SetVolume(v float32) error {
	s.volume = v
	s.setCalls++
	return nil
}

func (s *fakeSession) IsStale() bool {
	return s.stale
}

func (s *fakeSession) Release() {}

type fakeSessionFinder struct {
	sessions []Session
}

func (f *fakeSessionFinder) GetAllSessions() ([]Session, error) {
	return f.sessions, nil
}

func (f *fakeSessionFinder) Release() error {
	return nil
}

func TestHandleSliderMoveEventRetriesStaleSession(t *testing.T) {
	oldSession := &fakeSession{
		baseSession: baseSession{name: "firefox"},
		stale:       true,
	}
	newSession := &fakeSession{
		baseSession: baseSession{name: "firefox"},
	}

	config := &CanonicalConfig{SliderMapping: newSliderMap()}
	config.SliderMapping.set(1, []string{"firefox"})

	m := &sessionMap{
		deej:               &Deej{config: config},
		logger:             zap.NewNop().Sugar(),
		m:                  map[string][]Session{"firefox": {oldSession}},
		lock:               &sync.Mutex{},
		sessionFinder:      &fakeSessionFinder{sessions: []Session{newSession}},
		lastSessionRefresh: time.Now(),
	}

	m.handleSliderMoveEvent(SliderMoveEvent{SliderID: 1, PercentValue: 0.5})

	if oldSession.setCalls != 0 {
		t.Fatalf("stale session was adjusted %d time(s), want 0", oldSession.setCalls)
	}

	if newSession.setCalls != 1 {
		t.Fatalf("replacement session was adjusted %d time(s), want 1", newSession.setCalls)
	}

	if newSession.volume != 0.5 {
		t.Fatalf("replacement session volume = %v, want 0.5", newSession.volume)
	}
}
