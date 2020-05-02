package deej

import (
	"fmt"
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/mitchellh/go-ps"
	"github.com/moutend/go-wca"
	"github.com/omriharel/deej/util"
	"go.uber.org/zap"
)

// Session represents a single addressable audio session
type Session interface {
	GetVolume() float32
	SetVolume(v float32) error
	// GetMute() bool
	// SetMute(m bool) error

	Key() string
	Release()
}

type wcaSession struct {
	pid         uint32
	processName string
	system      bool

	logger  *zap.SugaredLogger
	control *wca.IAudioSessionControl2
	volume  *wca.ISimpleAudioVolume

	eventCtx *ole.GUID
}

func newWCASession(
	logger *zap.SugaredLogger,
	control *wca.IAudioSessionControl2,
	volume *wca.ISimpleAudioVolume,
	pid uint32,
	eventCtx *ole.GUID,
) (*wcaSession, error) {

	s := &wcaSession{
		control:  control,
		volume:   volume,
		pid:      pid,
		eventCtx: eventCtx,
	}

	// special treatment for system sounds session
	if pid == 0 {
		s.system = true
	} else {

		// find our session's process name
		process, err := ps.FindProcess(int(pid))
		if err != nil {
			logger.Warnw("Failed to find process name by ID", "pid", pid, "error", err)
			defer s.Release()

			return nil, err
		}

		s.processName = process.Executable()
	}

	// self-identifying session name e.g. deej.sessions.chrome
	s.logger = logger.Named(strings.TrimSuffix(s.Key(), ".exe"))
	s.logger.Debugw("Created audio session instance", "session", s)

	return s, nil
}

func (s *wcaSession) GetVolume() float32 {
	var level float32

	if err := s.volume.GetMasterVolume(&level); err != nil {
		s.logger.Warnw("Failed to get master volume", "error", err)
	}

	return util.NormalizeScalar(level)
}

func (s *wcaSession) SetVolume(v float32) error {
	if err := s.volume.SetMasterVolume(v, s.eventCtx); err != nil {
		s.logger.Warnw("Failed to set session volume", "error", err)
		return fmt.Errorf("adjust session volume: %w", err)
	}

	s.logger.Debugw("Adjusting session volume", "to", fmt.Sprintf("%.2f", v))

	return nil
}

func (s *wcaSession) Release() {
	s.logger.Debug("Releasing audio session")

	s.volume.Release()
	s.control.Release()
}

func (s *wcaSession) Key() string {
	if s.system {
		return systemSessionName
	}

	return strings.ToLower(s.processName)
}

func (s *wcaSession) String() string {
	sessionDesc := fmt.Sprintf("%s (pid %d)", s.processName, s.pid)

	if s.system {
		sessionDesc = "system sounds"
	}

	return fmt.Sprintf("<session: %s, vol: %.2f>", sessionDesc, s.GetVolume())
}
