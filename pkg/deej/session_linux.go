package deej

import (
	"errors"
	"fmt"
	"sync/atomic"

	"go.uber.org/zap"

	"github.com/jfreymuth/pulse/proto"
)

// normal PulseAudio volume (100%)
const maxVolume = 0x10000

var errNoSuchProcess = errors.New("No such process")

type paSession struct {
	baseSession

	processName string
	appName     string

	client *proto.Client

	sinkInputIndex    uint32
	sinkInputChannels byte

	// set when the sink input no longer exists, e.g. after the default output
	// changes. the session map refreshes and retries the adjustment when this
	// is observed.
	stale atomic.Bool
}

type masterSession struct {
	baseSession

	client *proto.Client

	streamIndex    uint32
	streamChannels byte
	isOutput       bool
}

func newPASession(
	logger *zap.SugaredLogger,
	client *proto.Client,
	sinkInputIndex uint32,
	sinkInputChannels byte,
	processName string,
	appName string,
) *paSession {

	s := &paSession{
		client:            client,
		sinkInputIndex:    sinkInputIndex,
		sinkInputChannels: sinkInputChannels,
	}

	s.processName = processName
	s.appName = appName
	s.name = processName
	s.humanReadableDesc = processName

	// use a self-identifying session name e.g. deej.sessions.chrome
	s.logger = logger.Named(s.Key())
	s.logger.Debugw(sessionCreationLogMessage, "session", s)

	return s
}

func newMasterSession(
	logger *zap.SugaredLogger,
	client *proto.Client,
	streamIndex uint32,
	streamChannels byte,
	isOutput bool,
) *masterSession {

	s := &masterSession{
		client:         client,
		streamIndex:    streamIndex,
		streamChannels: streamChannels,
		isOutput:       isOutput,
	}

	var key string

	if isOutput {
		key = masterSessionName
	} else {
		key = inputSessionName
	}

	s.logger = logger.Named(key)
	s.master = true
	s.name = key
	s.humanReadableDesc = key

	s.logger.Debugw(sessionCreationLogMessage, "session", s)

	return s
}

func (s *paSession) GetVolume() float32 {
	request := proto.GetSinkInputInfo{
		SinkInputIndex: s.sinkInputIndex,
	}
	reply := proto.GetSinkInputInfoReply{}

	if err := s.client.Request(&request, &reply); err != nil {
		s.stale.Store(true)
		s.logger.Debugw("Failed to get session volume, marking session stale", "error", err)
		return 0
	}

	s.stale.Store(false)

	level := parseChannelVolumes(reply.ChannelVolumes)

	return level
}

func (s *paSession) SetVolume(v float32) error {
	volumes := createChannelVolumes(s.sinkInputChannels, v)
	request := proto.SetSinkInputVolume{
		SinkInputIndex: s.sinkInputIndex,
		ChannelVolumes: volumes,
	}

	if err := s.client.Request(&request, nil); err != nil {
		s.stale.Store(true)
		s.logger.Debugw("Failed to set session volume, marking session stale", "error", err)
		return fmt.Errorf("adjust session volume: %w", err)
	}

	s.stale.Store(false)
	s.logger.Debugw("Adjusting session volume", "to", fmt.Sprintf("%.2f", v))

	return nil
}

func (s *paSession) IsStale() bool {
	return s.stale.Load()
}

func (s *paSession) Aliases() []string {
	return []string{s.processName, s.appName}
}

func (s *paSession) Release() {
	s.logger.Debug("Releasing audio session")
}

func (s *paSession) String() string {
	return fmt.Sprintf(sessionStringFormat, s.humanReadableDesc, s.GetVolume())
}

func (s *masterSession) GetVolume() float32 {
	var level float32

	// master follows the current default device, which can change when the
	// user plugs in headphones or switches outputs
	s.refreshDefault()

	if s.isOutput {
		request := proto.GetSinkInfo{
			SinkIndex: s.streamIndex,
		}
		reply := proto.GetSinkInfoReply{}

		if err := s.client.Request(&request, &reply); err != nil {
			s.logger.Warnw("Failed to get session volume", "error", err)
			return 0
		}

		level = parseChannelVolumes(reply.ChannelVolumes)
	} else {
		request := proto.GetSourceInfo{
			SourceIndex: s.streamIndex,
		}
		reply := proto.GetSourceInfoReply{}

		if err := s.client.Request(&request, &reply); err != nil {
			s.logger.Warnw("Failed to get session volume", "error", err)
			return 0
		}

		level = parseChannelVolumes(reply.ChannelVolumes)
	}

	return level
}

func (s *masterSession) SetVolume(v float32) error {
	var request proto.RequestArgs

	s.refreshDefault()

	volumes := createChannelVolumes(s.streamChannels, v)

	if s.isOutput {
		request = &proto.SetSinkVolume{
			SinkIndex:      s.streamIndex,
			ChannelVolumes: volumes,
		}
	} else {
		request = &proto.SetSourceVolume{
			SourceIndex:    s.streamIndex,
			ChannelVolumes: volumes,
		}
	}

	if err := s.client.Request(request, nil); err != nil {
		s.logger.Warnw("Failed to set session volume",
			"error", err,
			"volume", v)

		return fmt.Errorf("adjust session volume: %w", err)
	}

	s.logger.Debugw("Adjusting session volume", "to", fmt.Sprintf("%.2f", v))

	return nil
}

func (s *masterSession) refreshDefault() {
	if s.isOutput {
		request := proto.GetSinkInfo{
			SinkIndex: proto.Undefined,
		}
		reply := proto.GetSinkInfoReply{}

		if err := s.client.Request(&request, &reply); err != nil {
			s.logger.Warnw("Failed to get default sink info", "error", err)
			return
		}

		if s.streamIndex != reply.SinkIndex {
			s.logger.Debugw("Default sink changed",
				"oldIndex", s.streamIndex,
				"newIndex", reply.SinkIndex,
				"channels", reply.Channels)
		}

		s.streamIndex = reply.SinkIndex
		s.streamChannels = reply.Channels
		return
	}

	request := proto.GetSourceInfo{
		SourceIndex: proto.Undefined,
	}
	reply := proto.GetSourceInfoReply{}

	if err := s.client.Request(&request, &reply); err != nil {
		s.logger.Warnw("Failed to get default source info", "error", err)
		return
	}

	if s.streamIndex != reply.SourceIndex {
		s.logger.Debugw("Default source changed",
			"oldIndex", s.streamIndex,
			"newIndex", reply.SourceIndex,
			"channels", reply.Channels)
	}

	s.streamIndex = reply.SourceIndex
	s.streamChannels = reply.Channels
}

func (s *masterSession) Release() {
	s.logger.Debug("Releasing audio session")
}

func (s *masterSession) String() string {
	return fmt.Sprintf(sessionStringFormat, s.humanReadableDesc, s.GetVolume())
}

func createChannelVolumes(channels byte, volume float32) []uint32 {
	volumes := make([]uint32, channels)

	for i := range volumes {
		volumes[i] = uint32(volume * maxVolume)
	}

	return volumes
}

func parseChannelVolumes(volumes []uint32) float32 {
	if len(volumes) == 0 {
		return 0
	}

	var level uint32

	for _, volume := range volumes {
		level += volume
	}

	return float32(level) / float32(len(volumes)) / float32(maxVolume)
}
