#!/usr/bin/env python

import sys
import math
import time
import datetime
import os

import infi.systray
import serial
import yaml

from pycaw.pycaw import AudioUtilities, IAudioEndpointVolume
from ctypes import POINTER, cast
from comtypes import CLSCTX_ALL
from watchdog.events import FileSystemEventHandler
from watchdog.observers import Observer


class Deej(object):

    def __init__(self,):
        self._config_filename = 'config.yaml'
        self._config_directory = os.path.dirname(__file__)

        self._expected_num_sliders = None
        self._com_port = None
        self._baud_rate = None
        self._slider_values = None

        self._settings = None

        self._load_settings()

        self._sessions = None
        self._master_session = None

        self._devices = None

        self._last_session_refresh = None

        self._config_observer = None
        self._stopped = False

    def initialize(self):
        self._refresh_sessions()
        self._watch_config_file_changes()

    def stop(self):
        self._stopped = True

        if self._config_observer:
            self._config_observer.stop()

    def start(self):
        ser = serial.Serial()
        ser.baudrate = self._baud_rate
        ser.port = self._com_port
        ser.open()

        # ensure we start clean
        ser.readline()

        while not self._stopped:

            # read a single line from the serial stream, has values between 0 and 1023 separated by "|"
            line = ser.readline()

            # empty lines are a thing i guess
            if not line:
                try:
                    print 'Empty line'
                except:
                    pass

                continue

            # split on '|'
            split_line = line.split('|')


            if len(split_line) != self._expected_num_sliders:
                try:
                    print 'Uh oh - mismatch between number of sliders and config'
                except:
                    pass

                continue

            # now they're ints between 0 and 1023
            parsed_values = [int(n) for n in split_line]

            # now they're floats between 0 and 1 (but kinda dirty: 0.12334)
            normalized_values = [n / 1023.0 for n in parsed_values]

            # now they're cleaned up to 2 points of precision
            clean_values = [self._clean_session_volume(n) for n in normalized_values]

            if self._significantly_different_values(clean_values):
                self._slider_values = clean_values
                self._apply_volume_changes()

    def _load_settings(self, reload=False):
        settings = None

        try:
            with open(self._config_filename, 'rb') as f:
                raw_settings = f.read()
                settings = yaml.load(raw_settings, Loader=yaml.SafeLoader)
        except Exception as error:
            try:
                print 'Failed to {0}load config file {1}: {2}'.format('re' if reload else '',
                                                                    self._config_filename,
                                                                    error)
            except:
                pass

            if reload:
                return
            else:
                sys.exit(2)

        try:
            self._expected_num_sliders = len(settings['slider_mapping'])
            self._com_port = settings['com_port']
            self._baud_rate = settings['baud_rate']

            self._slider_values = [0] * self._expected_num_sliders

            self._settings = settings
        except Exception as error:
            try:
                print 'Failed to {0}load configuration, please ensure it matches' \
                    ' the required format. Error: {1}'.format('re' if reload else '', error)
            except:
                pass

        if reload:
            try:
                print 'Reloaded configuration successfully'
            except:
                pass

    def _watch_config_file_changes(self):

        class LogfileModifiedHandler(FileSystemEventHandler):

            @staticmethod
            def on_modified(event):
                if event.src_path.endswith(self._config_filename):
                    try:
                        print 'Detected config file changes, re-loading'
                    except:
                        pass

                    self._load_settings(reload=True)

        self._config_observer = Observer()
        self._config_observer.schedule(LogfileModifiedHandler(),
                                       self._config_directory,
                                       recursive=False)

        self._config_observer.start()

    def _refresh_sessions(self):

        # only do this if enough time passed since we last scanned for processes
        if self._last_session_refresh and time.time() - self._last_session_refresh < self._settings['process_refresh_frequency']:
            return

        self._last_session_refresh = time.time()

        # take all active sessions and map out those that belong to processes to their process name
        session_list = filter(lambda s: s.Process, AudioUtilities.GetAllSessions())
        self._sessions = {s.Process.name(): s for s in session_list}

        # take the master session
        active_device = AudioUtilities.GetSpeakers()
        active_device_interface = active_device.Activate(IAudioEndpointVolume._iid_, CLSCTX_ALL, None)
        self._master_session = cast(active_device_interface, POINTER(IAudioEndpointVolume))

    def _significantly_different_values(self, new_values):
        for idx, current_value in enumerate(self._slider_values):
            new_value = new_values[idx]

            if self._significantly_different_value(current_value, new_value):
                return True

        return False

    def _significantly_different_value(self, old, new):
        if abs(new - old) >= 0.025:
            return True

        # apply special behavior around the edges
        if (new == 1.0 and old != 1.0) or (new == 0.0 and old != 0.0):
            return True

        return False

    def _apply_volume_changes(self):
        for slider_idx, targets in self._settings['slider_mapping'].iteritems():

            slider_value = self._slider_values[slider_idx]
            target_found = False

            # normalize single target values
            if type(targets) is not list:
                targets = [targets]


            # first determine if the target is a currently active session:
            for target in targets:

                session_name, session = self._acquire_target_session(target)

                if session:
                    target_found = True

                    # get its current volume
                    current_volume = self._get_session_volume(session)

                    # set new one
                    if self._significantly_different_value(current_volume, slider_value):
                        self._set_session_volume(session, slider_value)

                        # if this fails while we're in the background - nobody cares!!!!!
                        try:
                            print '{0}: {1} => {2}'.format(session_name, current_volume, slider_value)
                        except:
                            pass

            # if we weren't able to find an audio session for this slider, maybe we aren't aware of that process yet
            if not target_found:
                self._refresh_sessions()

    def _acquire_target_session(self, name):
        if name == 'master':
            return 'Master', self._master_session

        for session_name, session in self._sessions.iteritems():
            if session_name.lower() == name.lower():
                return session_name, session

        return None, None

    def _get_session_volume(self, session):
        if hasattr(session, 'SimpleAudioVolume'):
            return self._clean_session_volume(session.SimpleAudioVolume.GetMasterVolume())

        return self._clean_session_volume(session.GetMasterVolumeLevelScalar())

    def _set_session_volume(self, session, value):
        if hasattr(session, 'SimpleAudioVolume'):
            session.SimpleAudioVolume.SetMasterVolume(value, None)
        else:
            session.SetMasterVolumeLevelScalar(value, None)

    def _clean_session_volume(self, value):
        return math.floor(value * 100) / 100.0


def setup_tray(stop_callback):
    tray = infi.systray.SysTrayIcon('assets/logo.ico', 'deej', on_quit=lambda _: stop_callback())
    tray.start()

    return tray

def main():
    deej = Deej()

    try:
        deej.initialize()
        tray = setup_tray(deej.stop)

        deej.start()

    except KeyboardInterrupt:
        print 'Interrupted.'
        sys.exit(130)
    except Exception as error:
        filename = 'deej-{0}.log'.format(datetime.datetime.now().strftime('%Y.%m.%d-%H.%M.%S'))

        with open(filename, 'w') as f:
            import traceback
            f.write('Exception occurred: {0}\nTraceback: {1}'.format(error, traceback.format_exc()))

        os.system('notepad.exe {0}'.format(filename))
        sys.exit(1)
    finally:
        tray.shutdown()
        print 'Bye!'


if __name__ == '__main__':
    main()
