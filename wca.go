package deej

import (
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca"

	"github.com/omriharel/deej/util"
)

func (s *sessionMap) getAllSessions() error {

	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		return err
	}
	defer ole.CoUninitialize()

	// get the IMMDeviceEnumerator
	var mmde *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
		return err
	}
	defer mmde.Release()

	// get the default audio endpoint as an IMMDevice
	var mmd *wca.IMMDevice
	if err := mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmd); err != nil {
		return err
	}
	defer mmd.Release()

	// query its IAudioSessionManager2 interface
	var asm2 *wca.IAudioSessionManager2
	if err := mmd.Activate(wca.IID_IAudioSessionManager2, wca.CLSCTX_ALL, nil, &asm2); err != nil {
		return err
	}
	defer asm2.Release()

	// get its IAudioSessionEnumerator
	var sessionEnumerator *wca.IAudioSessionEnumerator
	if err := asm2.GetSessionEnumerator(&sessionEnumerator); err != nil {
		return err
	}
	defer sessionEnumerator.Release()

	// enumerate its sessions
	var sessionCount int
	if err := sessionEnumerator.GetCount(&sessionCount); err != nil {
		return err
	}

	s.logger.Debugf("Seeing %d sessions", sessionCount)

	// for each session, get its stuff
	for sessionIdx := 0; sessionIdx < sessionCount; sessionIdx++ {

		// get the IAudioSessionControl
		var session *wca.IAudioSessionControl
		if err := sessionEnumerator.GetSession(sessionIdx, &session); err != nil {
			return err
		}

		// query its IAudioSessionControl2
		controlDispatch, err := session.QueryInterface(wca.IID_IAudioSessionControl2)
		if err != nil {
			return err
		}

		session.Release()

		control := (*wca.IAudioSessionControl2)(unsafe.Pointer(controlDispatch))
		defer control.Release()

		var pid uint32
		if err := control.GetProcessId(&pid); err != nil {

			// silently ignore the system sounds session - maybe we'll support it too at some point
			if control.IsSystemSoundsSession() != nil {
				return err
			}

			continue
		}

		volumeDispatch, err := control.QueryInterface(wca.IID_ISimpleAudioVolume)
		if err != nil {
			return err
		}

		volume := (*wca.ISimpleAudioVolume)(unsafe.Pointer(volumeDispatch))
		defer volume.Release()

		var level float32
		if err := volume.GetMasterVolume(&level); err != nil {
			return err
		}

		s.logger.Debugf("Got audio session for PID %d - volume level is %.2f", pid, util.NormalizeScalar(level))
	}

	return nil
}
