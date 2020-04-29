#!/usr/bin/env python

import sys
import math
import time
import datetime
import os
import subprocess
import psutil

import infi.systray
import serial
import yaml

from pycaw.pycaw import AudioUtilities, IAudioEndpointVolume
from ctypes import POINTER, pointer, cast
from comtypes import CLSCTX_ALL, GUID
from watchdog.events import FileSystemEventHandler
from watchdog.observers import Observer


class Deej(object):

    def __init__(self,):
        self._config_filename = 'config.yaml'
        self._config_directory = os.path.dirname(os.path.abspath(__file__))

        self._expected_num_sliders = None
        self._com_port = None
        self._baud_rate = None
        self._slider_values = None

        self._settings = None

        self._load_settings()

        self._sessions = None
        self._master_session = None

        self._devices = None

        self._should_refresh_sessions = False
        self._last_session_refresh = None

        self._config_observer = None
        self._stopped = False

        self._lpcguid = pointer(GUID.create_new())

    def initialize(self):
        self._refresh_sessions()
        self._watch_config_file_changes()

    def stop(self):
        self._stopped = True

        if self._config_observer:
            self._config_observer.stop()

    def edit_config(self):
        attempt_print('Opening config file for editing')
        spawn_detached_notepad(self._config_filename)

    def queue_session_refresh(self):
        self._should_refresh_sessions = True

    def start(self):
        ser = serial.Serial()
        ser.baudrate = self._baud_rate
        ser.port = self._com_port
        ser.open()

        # ensure we start clean
        ser.readline()

        while not self._stopped:

            # check if the user requested to refresh sessions
            if self._should_refresh_sessions:
                self._refresh_sessions()

            # read a single line from the serial stream, has values between 0 and 1023 separated by "|"
            line = ser.readline()

            # empty lines are a thing i guess
            if not line:
                attempt_print('Empty line')
                continue

            # split on '|'
            split_line = line.split('|')


            if len(split_line) != self._expected_num_sliders:
                attempt_print('Uh oh - mismatch between number of sliders and config')
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
            attempt_print('Failed to {0}load config file {1}: {2}'.format('re' if reload else '',
                                                                          self._config_filename,
                                                                          error))

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
            attempt_print('Failed to {0}load configuration, please ensure it matches' \
                ' the required format. Error: {1}'.format('re' if reload else '', error))

        if reload:
            attempt_print('Reloaded configuration successfully')

    def _watch_config_file_changes(self):

        class LogfileModifiedHandler(FileSystemEventHandler):

            @staticmethod
            def on_modified(event):
                if event.src_path.endswith(self._config_filename):
                    attempt_print('Detected config file changes, re-loading')
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
        self._sessions = {}

        for session in session_list:
            try:
                session_name = session.Process.name()
            except psutil.NoSuchProcess:
                continue

            if session_name in self._sessions:
                self._sessions[session_name].append(session)
            else:
                self._sessions[session_name] = [session]

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

                sessions = self._acquire_target_sessions(target)

                # mark target as resolved
                if sessions:
                    target_found = True

                    # each target may resolve to multiple sessions. for each such session:
                    for session_name, session in sessions:

                        # get its current volume
                        current_volume = self._get_session_volume(session)

                        # set new one
                        if self._significantly_different_value(current_volume, slider_value):
                            self._set_session_volume(session, slider_value)

                            # if this fails while we're in the background - nobody cares!!!!!
                            attempt_print('{0}: {1} => {2}'.format(session_name, current_volume, slider_value))


            # if we weren't able to find an audio session for this slider,
            # maybe we aren't aware of that process yet. better check
            if not target_found:
                self._refresh_sessions()

    def _acquire_target_sessions(self, name):

        if name == 'master':
            return [('Master', self._master_session)]

        for process_name, process_sessions in self._sessions.iteritems():
            if process_name.lower() == name.lower():

                # if we only have one session for that process return it
                if len(process_sessions) == 1:
                    return [(process_name, process_sessions[0])]

                # if we have more, number them for logging and stuff
                return [('{0} ({1})'.format(process_name, idx), session) for idx, session in enumerate(process_sessions)]

        return []

    def _get_session_volume(self, session):
        if hasattr(session, 'SimpleAudioVolume'):
            return self._clean_session_volume(session.SimpleAudioVolume.GetMasterVolume())

        return self._clean_session_volume(session.GetMasterVolumeLevelScalar())

    def _set_session_volume(self, session, value):
        if hasattr(session, 'SimpleAudioVolume'):
            session.SimpleAudioVolume.SetMasterVolume(value, self._lpcguid)
        else:
            session.SetMasterVolumeLevelScalar(value, self._lpcguid)

    def _clean_session_volume(self, value):
        return math.floor(value * 100) / 100.0


def setup_tray(edit_config_callback, refresh_sessions_callback, stop_callback):
    menu_options = (('Edit configuration', None, lambda _: edit_config_callback()),
                    ('Re-scan audio sessions', None, lambda _: refresh_sessions_callback()))

    tray = infi.systray.SysTrayIcon('assets/logo.ico', 'deej', menu_options, on_quit=lambda _: stop_callback())
    tray.start()

    return tray


def attempt_print(s):
    try:
        print s
    except:
        pass


def spawn_detached_notepad(filename):
    subprocess.Popen(['notepad.exe', filename],
                     close_fds=True,
                     creationflags=0x00000008)


def main():
    deej = Deej()

    try:
        deej.initialize()
        tray = setup_tray(deej.edit_config, deej.queue_session_refresh, deej.stop)

        deej.start()

    except KeyboardInterrupt:
        attempt_print('Interrupted.')
        sys.exit(130)
    except Exception as error:
        filename = 'deej-{0}.log'.format(datetime.datetime.now().strftime('%Y.%m.%d-%H.%M.%S'))

        with open(filename, 'w') as f:
            import traceback
            f.write('Unfortunately, deej has crashed. This really shouldn\'t happen!\n')
            f.write('If you\'ve just encountered this, please contact @omriharel and attach this error log.\n')
            f.write('You can also join the deej Discord server at https://discord.gg/nf88NJu.\n')
            f.write('Exception occurred: {0}\nTraceback: {1}'.format(error, traceback.format_exc()))

        spawn_detached_notepad(filename)
        sys.exit(1)
    finally:
        tray.shutdown()
        attempt_print('Bye!')


if __name__ == '__main__':
    main()
