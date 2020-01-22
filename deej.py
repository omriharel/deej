#!/usr/bin/env python

import sys
import math
import time
import serial

from pycaw.pycaw import AudioUtilities, IAudioEndpointVolume
from ctypes import POINTER, cast
from comtypes import CLSCTX_ALL


class Deej(object):

    def __init__(self, settings):
        self._settings = settings

        self._sessions = None
        self._master_session = None

        self._devices = None

        self._expected_num_sliders = len(settings['slider_mapping'])

        self._slider_values = [0] * self._expected_num_sliders

        self._last_session_refresh = None

    def initialize(self):
        self._refresh_sessions()

    def accept_commands(self):
        ser = serial.Serial()
        ser.baudrate = 9600
        ser.port = 'COM3'
        ser.open()

        # ensure we start clean
        ser.readline()

        while True:

            # read a single line from the serial stream, has values between 0 and 1023 separated by "|"
            line = ser.readline()

            # empty lines are a thing i guess
            if not line:
                print 'Empty line'
                continue

            # split on '|'
            split_line = line.split('|')


            if len(split_line) != self._expected_num_sliders:
                print 'Uh oh - mismatch between number of sliders and config'

            # now they're ints between 0 and 1023
            parsed_values = [int(n) for n in split_line]

            # now they're floats between 0 and 1 (but kinda dirty: 0.12334)
            normalized_values = [n / 1023.0 for n in parsed_values]

            # now they're cleaned up to 2 points of precision
            clean_values = [self._clean_session_volume(n) for n in normalized_values]

            if self._significantly_different_values(clean_values):
                self._slider_values = clean_values
                self._apply_volume_changes()

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
                        print '{0}: {1} => {2}'.format(session_name, current_volume, slider_value)

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


def main():
    deej = Deej({
        'slider_mapping': {
            0: 'master',
            1: 'chrome.exe',
            2: 'spotify.exe',
            3: [
                'pathofexile_x64.exe',
                'rocketleague.exe',
            ],
            4: 'discord.exe',
        },
        'process_refresh_frequency': 5,
    })

    try:
        deej.initialize()
        deej.accept_commands()

    except KeyboardInterrupt:
        print 'Bye!'
        sys.exit(130)


if __name__ == '__main__':
    main()
