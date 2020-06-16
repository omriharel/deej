package deej

import (
	"errors"
	"fmt"
	"strings"

	ole "github.com/go-ole/go-ole"
	ps "github.com/mitchellh/go-ps"
	wca "github.com/moutend/go-wca"
	"go.uber.org/zap"
)

var errNoSuchProcess = errors.New("No such process")
var errRefreshSessions = errors.New("Trigger session refresh")

// WcaSession sub session controllers
type WcaSession struct {
	baseSession

	pid         uint32
	processName string

	control *wca.IAudioSessionControl2
	volume  *wca.ISimpleAudioVolume

	eventCtx *ole.GUID
}

// MasterSession master session controllers
type MasterSession struct {
	baseSession

	volume *wca.IAudioEndpointVolume

	eventCtx *ole.GUID

	stale bool // when set to true, we should refresh sessions on the next call to SetVolume
}

func newWCASession(
	logger *zap.SugaredLogger,
	control *wca.IAudioSessionControl2,
	volume *wca.ISimpleAudioVolume,
	pid uint32,
	eventCtx *ole.GUID,
) (*WcaSession, error) {

	s := &WcaSession{
		control:  control,
		volume:   volume,
		pid:      pid,
		eventCtx: eventCtx,
	}

	// special treatment for system sounds session
	if pid == 0 {
		s.system = true
		s.name = systemSessionName
		s.humanReadableDesc = "system sounds"
	} else {

		// find our session's process name
		process, err := ps.FindProcess(int(pid))
		if err != nil {
			logger.Warnw("Failed to find process name by ID", "pid", pid, "error", err)
			defer s.Release()

			return nil, fmt.Errorf("find process name by pid: %w", err)
		}

		// this PID may be invalid - this means the process has already been
		// closed and we shouldn't create a session for it.
		if process == nil {
			logger.Debugw("Process already exited, not creating audio session", "pid", pid)
			return nil, errNoSuchProcess
		}

		s.processName = process.Executable()
		s.name = s.processName
		s.humanReadableDesc = fmt.Sprintf("%s (pid %d)", s.processName, s.pid)
	}

	// use a self-identifying session name e.g. deej.sessions.chrome
	s.logger = logger.Named(strings.TrimSuffix(s.Key(), ".exe"))
	s.logger.Debugw(sessionCreationLogMessage, "session", s)
	return s, nil
}

func newMasterSession(
	logger *zap.SugaredLogger,
	volume *wca.IAudioEndpointVolume,
	eventCtx *ole.GUID,
	key string,
) (*MasterSession, error) {

	s := &MasterSession{
		volume:   volume,
		eventCtx: eventCtx,
	}

	s.logger = logger.Named(key)
	s.master = true
	s.name = key
	s.humanReadableDesc = key

	s.logger.Debugw(sessionCreationLogMessage, "session", s)

	return s, nil
}

// GetIconPath returns the icon path for the session
func (s *WcaSession) GetIconPath() string {
	var iconPath string
	if err := s.control.GetIconPath(&iconPath); err != nil {
		s.logger.Warnw("Failed to get sessiom icon", "error", err)
	}
	return iconPath
}

// GetVolume returns the volume of the session
func (s *WcaSession) GetVolume() float32 {
	var level float32

	if err := s.volume.GetMasterVolume(&level); err != nil {
		s.logger.Warnw("Failed to get session volume", "error", err)
	}

	return level
}

// SetVolume sets the volume of the session
func (s *WcaSession) SetVolume(v float32) error {
	if err := s.volume.SetMasterVolume(v, s.eventCtx); err != nil {
		s.logger.Warnw("Failed to set session volume", "error", err)
		return fmt.Errorf("adjust session volume: %w", err)
	}

	// mitigate expired sessions by checking the state whenever we change volumes
	var state uint32

	if err := s.control.GetState(&state); err != nil {
		s.logger.Warnw("Failed to get session state while setting volume", "error", err)
		return fmt.Errorf("get session state: %w", err)
	}

	if state == wca.AudioSessionStateExpired {
		s.logger.Warnw("Audio session expired, triggering session refresh")
		return errRefreshSessions
	}

	s.logger.Debugw("Adjusting session volume", "to", fmt.Sprintf("%.2f", v))

	return nil
}

// Release relases the session parts to the wind
func (s *WcaSession) Release() {
	s.logger.Debug("Releasing audio session")

	s.volume.Release()
	s.control.Release()
}

func (s *WcaSession) String() string {
	return fmt.Sprintf(sessionStringFormat, s.humanReadableDesc, s.GetVolume())
}

// GetVolume returns the volume of the master session
func (s *MasterSession) GetVolume() float32 {
	var level float32

	if err := s.volume.GetMasterVolumeLevelScalar(&level); err != nil {
		s.logger.Warnw("Failed to get session volume", "error", err)
	}

	return level
}

// SetVolume sets the volume of the master sessions
func (s *MasterSession) SetVolume(v float32) error {
	if s.stale {
		s.logger.Warnw("Session expired because default device has changed, triggering session refresh")
		return errRefreshSessions
	}

	if err := s.volume.SetMasterVolumeLevelScalar(v, s.eventCtx); err != nil {
		s.logger.Warnw("Failed to set session volume",
			"error", err,
			"volume", v)

		return fmt.Errorf("adjust session volume: %w", err)
	}

	s.logger.Debugw("Adjusting session volume", "to", fmt.Sprintf("%.2f", v))

	return nil
}

// Release relases the master session parts to the wind
func (s *MasterSession) Release() {
	s.logger.Debug("Releasing audio session")

	s.volume.Release()
}

func (s *MasterSession) String() string {
	return fmt.Sprintf(sessionStringFormat, s.humanReadableDesc, s.GetVolume())
}

func (s *MasterSession) markAsStale() {
	s.stale = true
}
