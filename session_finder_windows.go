package deej

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"

	ole "github.com/go-ole/go-ole"
	wca "github.com/moutend/go-wca"
	"go.uber.org/zap"
)

type wcaSessionFinder struct {
	logger        *zap.SugaredLogger
	sessionLogger *zap.SugaredLogger

	eventCtx *ole.GUID // needed for some session actions to successfully notify other audio consumers
}

const (

	// there's no real mystery here, it's just a random GUID
	myteriousGUID = "{1ec920a1-7db8-44ba-9779-e5d28ed9f330}"
)

func newSessionFinder(logger *zap.SugaredLogger) (SessionFinder, error) {
	sf := &wcaSessionFinder{
		logger:        logger.Named("session_finder"),
		sessionLogger: logger.Named("sessions"),
		eventCtx:      ole.NewGUID(myteriousGUID),
	}

	sf.logger.Debug("Created WCA session finder instance")

	return sf, nil
}

func (sf *wcaSessionFinder) GetAllSessions() ([]Session, error) {
	sessions := []Session{}

	// we must call this every time we're about to list devices, i think. could be wrong
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		sf.logger.Warnw("Failed to call CoInitializeEx", "error", err)
		return nil, fmt.Errorf("call CoInitializeEx: %w", err)
	}
	defer ole.CoUninitialize()

	// get the active device
	defaultAudioEndpoint, err := getDefaultAudioEndpoint()
	if err != nil {
		sf.logger.Warnw("Failed to get default audio endpoint", "error", err)
		return nil, fmt.Errorf("get default audio endpoint: %w", err)
	}
	defer defaultAudioEndpoint.Release()

	// get the master session
	master, err := sf.getMasterSession(defaultAudioEndpoint)
	if err != nil {
		sf.logger.Warnw("Failed to get master audio session", "error", err)
		return nil, fmt.Errorf("get master audio session: %w", err)
	}

	sessions = append(sessions, master)

	// get an enumerator for the rest of the sessions
	sessionEnumerator, err := getSessionEnumerator(defaultAudioEndpoint)
	if err != nil {
		sf.logger.Warnw("Failed to get audio session enumerator", "error", err)
		return nil, fmt.Errorf("get audio session enumerator: %w", err)
	}
	defer sessionEnumerator.Release()

	// enumerate it and add sessions along the way
	if err := sf.enumerateAndAddSessions(sessionEnumerator, &sessions); err != nil {
		sf.logger.Warnw("Failed to enumerate audio sessions", "error", err)
		return nil, fmt.Errorf("enumerate audio sessions: %w", err)
	}

	return sessions, nil
}

func (sf *wcaSessionFinder) Release() error {
	sf.logger.Debug("Released WCA session finder instance")

	return nil
}

func getDefaultAudioEndpoint() (*wca.IMMDevice, error) {

	// get the IMMDeviceEnumerator
	var mmDeviceEnumerator *wca.IMMDeviceEnumerator

	if err := wca.CoCreateInstance(
		wca.CLSID_MMDeviceEnumerator,
		0,
		wca.CLSCTX_ALL,
		wca.IID_IMMDeviceEnumerator,
		&mmDeviceEnumerator,
	); err != nil {
		return nil, err
	}
	defer mmDeviceEnumerator.Release()

	// get the default audio endpoint as an IMMDevice
	var mmDevice *wca.IMMDevice

	if err := mmDeviceEnumerator.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmDevice); err != nil {
		return nil, err
	}

	return mmDevice, nil
}

func (sf *wcaSessionFinder) getMasterSession(mmDevice *wca.IMMDevice) (Session, error) {

	var audioEndpointVolume *wca.IAudioEndpointVolume

	if err := mmDevice.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &audioEndpointVolume); err != nil {
		sf.logger.Warnw("Failed to activate AudioEndpointVolume for master session", "error", err)
		return nil, fmt.Errorf("activate master session: %w", err)
	}

	// create the master session
	master, err := newMasterSession(sf.sessionLogger, audioEndpointVolume, sf.eventCtx)
	if err != nil {
		sf.logger.Warnw("Failed to create master session instance", "error", err)
		return nil, fmt.Errorf("create master session: %w", err)
	}

	return master, nil
}

func getSessionEnumerator(mmDevice *wca.IMMDevice) (*wca.IAudioSessionEnumerator, error) {

	// query the given IMMDevice's IAudioSessionManager2 interface
	var audioSessionManager2 *wca.IAudioSessionManager2

	if err := mmDevice.Activate(
		wca.IID_IAudioSessionManager2,
		wca.CLSCTX_ALL,
		nil,
		&audioSessionManager2,
	); err != nil {
		return nil, err
	}
	defer audioSessionManager2.Release()

	// get its IAudioSessionEnumerator
	var audioSessionEnumerator *wca.IAudioSessionEnumerator

	if err := audioSessionManager2.GetSessionEnumerator(&audioSessionEnumerator); err != nil {
		return nil, err
	}

	return audioSessionEnumerator, nil
}

func (sf *wcaSessionFinder) enumerateAndAddSessions(
	sessionEnumerator *wca.IAudioSessionEnumerator,
	sessions *[]Session,
) error {

	// check how many audio sessions there are
	var sessionCount int

	if err := sessionEnumerator.GetCount(&sessionCount); err != nil {
		sf.logger.Warnw("Failed to get session count from session enumerator", "error", err)
		return fmt.Errorf("get session count: %w", err)
	}

	sf.logger.Debugw("Got session count from session enumerator", "count", sessionCount)

	// for each session:
	for sessionIdx := 0; sessionIdx < sessionCount; sessionIdx++ {

		// get the IAudioSessionControl
		var audioSessionControl *wca.IAudioSessionControl
		if err := sessionEnumerator.GetSession(sessionIdx, &audioSessionControl); err != nil {
			sf.logger.Warnw("Failed to get session from session enumerator",
				"error", err,
				"sessionIdx", sessionIdx)

			return fmt.Errorf("get session %d from enumerator: %w", sessionIdx, err)
		}

		// query its IAudioSessionControl2
		dispatch, err := audioSessionControl.QueryInterface(wca.IID_IAudioSessionControl2)
		if err != nil {
			sf.logger.Warnw("Failed to query session's IAudioSessionControl2",
				"error", err,
				"sessionIdx", sessionIdx)

			return fmt.Errorf("query session %d IAudioSessionControl2: %w", sessionIdx, err)
		}

		// we no longer need the IAudioSessionControl, release it
		audioSessionControl.Release()

		// receive a useful object instead of our dispatch
		audioSessionControl2 := (*wca.IAudioSessionControl2)(unsafe.Pointer(dispatch))

		var pid uint32

		// get the session's PID
		if err := audioSessionControl2.GetProcessId(&pid); err != nil {

			// if this is the system sounds session, GetProcessId will error with an undocumented
			// AUDCLNT_S_NO_CURRENT_PROCESS (0x889000D) - this is fine, we actually want to treat it a bit differently
			// The first part of this condition will be true if the call to IsSystemSoundsSession fails
			// The second part will be true if the original error mesage from GetProcessId doesn't contain this magical
			// error code (in decimal format).
			isSystemSoundsErr := audioSessionControl2.IsSystemSoundsSession()
			if isSystemSoundsErr != nil && !strings.Contains(err.Error(), "143196173") {

				// of course, if it's not the system sounds session, we got a problem
				sf.logger.Warnw("Failed to query session's pid",
					"error", err,
					"isSystemSoundsError", isSystemSoundsErr,
					"sessionIdx", sessionIdx)

				return fmt.Errorf("query session %d pid: %w", sessionIdx, err)
			}

			// make sure to indicate that this is a system sounds session
			pid = 0
		}

		// get its ISimpleAudioVolume
		dispatch, err = audioSessionControl2.QueryInterface(wca.IID_ISimpleAudioVolume)
		if err != nil {
			sf.logger.Warnw("Failed to query session's ISimpleAudioVolume",
				"error", err,
				"sessionIdx", sessionIdx)

			return fmt.Errorf("query session %d ISimpleAudioVolume: %w", sessionIdx, err)
		}

		// make it useful, again
		simpleAudioVolume := (*wca.ISimpleAudioVolume)(unsafe.Pointer(dispatch))

		// create the deej session object
		newSession, err := newWCASession(sf.sessionLogger, audioSessionControl2, simpleAudioVolume, pid, sf.eventCtx)
		if err != nil {

			// this could just mean this process is already closed by now, and the session will be cleaned up later by the OS
			if !errors.Is(err, errNoSuchProcess) {
				sf.logger.Warnw("Failed to create new WCA session instance",
					"error", err,
					"sessionIdx", sessionIdx)

				return fmt.Errorf("create wca session for session %d: %w", sessionIdx, err)
			}

			// in this case, log it and release the session's handles, then skip to the next one
			sf.logger.Debugw("Process already exited, skipping session and releasing handles", "pid", pid)

			audioSessionControl2.Release()
			simpleAudioVolume.Release()

			continue
		}

		// add it to our slice
		*sessions = append(*sessions, newSession)
	}

	return nil
}
