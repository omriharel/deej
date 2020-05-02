package deej

import (
	"github.com/moutend/go-wca"
	"go.uber.org/zap"
)

// Session represents a single addressable audio session
type Session interface {
	GetVolume() float32
	// SetVolume(v float32) error
	// GetMute() bool
	// SetMute(m bool) error
}

type wcaSession struct {
	pid         uint32
	processName string
	master      bool

	logger  *zap.SugaredLogger
	control *wca.IAudioSessionControl2
	volume  *wca.ISimpleAudioVolume
}

func newWCASession(logger *zap.SugaredLogger, control *wca.IAudioSessionControl2, volume *wca.ISimpleAudioVolume) (*wcaSession, error) {
	s := &wcaSession{
		logger:  logger.Named("session"),
		control: control,
		volume:  volume,
	}

	// TODO set PID/process name/master state here

	return s, nil
}

func (s *wcaSession) GetVolume() float32 {
	var level float32

	if err := s.volume.GetMasterVolume(&level); err != nil {
		s.logger.Warnw("Failed to get master volume", "error", err)
	}

	return level
}
