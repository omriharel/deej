package deej

import (
	"fmt"
	"time"
	"unsafe"

	ole "github.com/go-ole/go-ole"
	wca "github.com/moutend/go-wca"
)

func (m *sessionMap) getAllSessions() error {

	// we must call this every time we're about to list devices, i think. could be wrong
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		m.logger.Warnw("Failed to call CoInitializeEx", "error", err)
		return fmt.Errorf("call CoInitializeEx: %w", err)
	}
	defer ole.CoUninitialize()

	// get the active device
	defaultAudioEndpoint, err := getDefaultAudioEndpoint()
	if err != nil {
		m.logger.Warnw("Failed to get default audio endpoint", "error", err)
		return fmt.Errorf("get default audio endpoint: %w", err)
	}
	defer defaultAudioEndpoint.Release()

	// get the master session
	if err := m.getAndAddMasterSession(defaultAudioEndpoint); err != nil {
		m.logger.Warnw("Failed to get master audio session", "error", err)
		return fmt.Errorf("get master audio session: %w", err)
	}

	// get an enumerator for the rest of the sessions
	sessionEnumerator, err := getSessionEnumerator(defaultAudioEndpoint)
	if err != nil {
		m.logger.Warnw("Failed to get audio session enumerator", "error", err)
		return fmt.Errorf("get audio session enumerator: %w", err)
	}
	defer sessionEnumerator.Release()

	// enumerate it and add sessions along the way
	if err := m.enumerateAndAddSessions(sessionEnumerator); err != nil {
		m.logger.Warnw("Failed to enumerate audio sessions", "error", err)
		return fmt.Errorf("enumerate audio sessions: %w", err)
	}

	m.logger.Debugw("Got all audio sessions successfully", "sessionMap", m)

	// mark completion
	m.lastSessionRefresh = time.Now()

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

func (m *sessionMap) getAndAddMasterSession(mmDevice *wca.IMMDevice) error {

	var audioEndpointVolume *wca.IAudioEndpointVolume

	if err := mmDevice.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &audioEndpointVolume); err != nil {
		m.logger.Warnw("Failed to activate AudioEndpointVolume for master session", "error", err)
		return fmt.Errorf("activate master session: %w", err)
	}

	// create the master session
	master, err := newMasterSession(m.logger, audioEndpointVolume, m.eventCtx)
	if err != nil {
		m.logger.Warnw("Failed to create master session instance", "error", err)
		return fmt.Errorf("create master session: %w", err)
	}

	m.add(master)

	return nil
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

func (m *sessionMap) enumerateAndAddSessions(sessionEnumerator *wca.IAudioSessionEnumerator) error {

	// check how many audio sessions there are
	var sessionCount int

	if err := sessionEnumerator.GetCount(&sessionCount); err != nil {
		m.logger.Warnw("Failed to get session count from session enumerator", "error", err)
		return fmt.Errorf("get session count: %w", err)
	}

	m.logger.Debugw("Got session count from session enumerator", "count", sessionCount)

	// for each session:
	for sessionIdx := 0; sessionIdx < sessionCount; sessionIdx++ {

		// get the IAudioSessionControl
		var audioSessionControl *wca.IAudioSessionControl
		if err := sessionEnumerator.GetSession(sessionIdx, &audioSessionControl); err != nil {
			m.logger.Warnw("Failed to get session from session enumerator",
				"error", err,
				"sessionIdx", sessionIdx)

			return fmt.Errorf("get session %d from enumerator: %w", sessionIdx, err)
		}

		// query its IAudioSessionControl2
		dispatch, err := audioSessionControl.QueryInterface(wca.IID_IAudioSessionControl2)
		if err != nil {
			m.logger.Warnw("Failed to query session's IAudioSessionControl2",
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
			if audioSessionControl2.IsSystemSoundsSession() == nil {
				// system sounds session
			} else {

				// of course, if it's not the system sounds session, we got a problem
				m.logger.Warnw("Failed to query session's pid",
					"error", err,
					"sessionIdx", sessionIdx)

				return fmt.Errorf("query session %d pid: %w", sessionIdx, err)
			}
		}

		// get its ISimpleAudioVolume
		dispatch, err = audioSessionControl2.QueryInterface(wca.IID_ISimpleAudioVolume)
		if err != nil {
			m.logger.Warnw("Failed to query session's ISimpleAudioVolume",
				"error", err,
				"sessionIdx", sessionIdx)

			return fmt.Errorf("query session %d ISimpleAudioVolume: %w", sessionIdx, err)
		}

		// make it useful, again
		simpleAudioVolume := (*wca.ISimpleAudioVolume)(unsafe.Pointer(dispatch))

		// create the deej session object
		newSession, err := newWCASession(m.logger, audioSessionControl2, simpleAudioVolume, pid, m.eventCtx)
		if err != nil {
			m.logger.Warnw("Failed to create new WCA session instance",
				"error", err,
				"sessionIdx", sessionIdx)

			return fmt.Errorf("create wca session for session %d: %w", sessionIdx, err)
		}

		// add it to our map
		m.add(newSession)
	}

	return nil
}
