package hls

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ErrNoAudioInQueue is an error stating that there are no more audio to stream in the queue
var ErrNoAudioInQueue = errors.New("no audio in queue")

const (
	// DefaultEncoding is the default encoding used with ffmpeg
	DefaultEncoding = "aac"
	// DefaultHLSListSize is the default hls_list_size for ffmpeg
	DefaultHLSListSize = 80
	// DefaultHLSTime is the default hls_time for ffmpeg
	DefaultHLSTime = 5
	// DefaultSegmentFilename is the default hls_segment_filename for ffmpeg
	DefaultSegmentFilename = "hls-%v/hls-segment-%06d.ts"
	// DefaultPlaylistName is the default output name for ffmpeg
	DefaultPlaylistName = "hls-%v/hls-playlist.m3u8"
	// DefaultMasterPlaylistName is the default master_pl_name for ffmpeg
	DefaultMasterPlaylistName = "master.m3u8"
	// DefaultDequeuedTimeout is the default read timeout for the dequeued notification channel
	DefaultDequeuedTimeout = time.Second
	// DefaultIsDebugging is the default debugging option
	DefaultIsDebugging = false
)

// Stream is responsible for streaming the audio
type Stream struct {
	// Options
	encoding           string
	bitrates           []string
	hlsListSize        int
	hlsTime            int
	segmentFilename    string
	playlistName       string
	masterPlaylistName string
	dequeuedTimeout    time.Duration
	isLogging          bool
	setMetadata        func(Audio) string

	queue    []Audio
	dequeued chan Audio
	buffer   *bytes.Buffer
	audioMux *sync.Mutex
}

// Audio is to be streamed
type Audio struct {
	Data             io.Reader
	ID               int64
	Artist, Title    string
	Metadata         map[string]string
	OverrideEncoding string
}

func (a Audio) String() string {
	return a.Artist + " - " + a.Title
}

// NewStream initializes and returns a Stream
func NewStream(bitrates []string, opts ...StreamOption) (Stream, error) {
	s := &Stream{
		bitrates: bitrates,

		// Options
		encoding:           DefaultEncoding,
		hlsListSize:        DefaultHLSListSize,
		hlsTime:            DefaultHLSTime,
		segmentFilename:    DefaultSegmentFilename,
		playlistName:       DefaultPlaylistName,
		masterPlaylistName: DefaultMasterPlaylistName,
		dequeuedTimeout:    DefaultDequeuedTimeout,
		isLogging:          DefaultIsDebugging,
		setMetadata:        defaultMetadataTitle,

		queue:    []Audio{},
		dequeued: make(chan Audio),
		audioMux: &sync.Mutex{},
		buffer:   bytes.NewBuffer([]byte{}),
	}

	for _, o := range opts {
		o(s)
	}

	if filepath.Dir(s.masterPlaylistName) != "." {
		return Stream{}, errors.New("master playlist cannot have a directory, it must be root. ffmpeg limitation")
	}

	return *s, nil
}

// Append will append an Audio in the back of the queue, it will to be streamed
func (s *Stream) Append(a Audio) {
	s.audioMux.Lock()
	s.queue = append(s.queue, a)
	s.log("queue size increased to %d", len(s.queue))
	s.audioMux.Unlock()
}

// QueueSize returns the current size of the queue
func (s *Stream) QueueSize() int {
	return len(s.queue)
}

// Dequeued will notify when an Audio has been dequeue and is currently being written
// into the HLS format. It is optional to read this channel, "messages" will be dropped after
// the timeout which is configurable with the option WithDequeuedTimeout
func (s *Stream) Dequeued() <-chan Audio {
	return s.dequeued
}

// Start will start the HLS convertion and write the files based on the configuration. This is a blocking call.
// Check out the Options and defaults which will have an impact on the convertion.
func (s *Stream) Start() error {
	s.log("started to stream")
	if err := s.execFFmpeg(); err != nil {
		s.log("failed to execute ffmpeg command: %s", err)
		return err
	}

	return nil
}

func (s *Stream) execFFmpeg() error {
	var (
		hlsSize     = strconv.Itoa(s.hlsListSize)
		hlsTime     = strconv.Itoa(s.hlsTime)
		bitrateArgs = bitratesToArgs(s.bitrates)
	)

	for {
		a, err := s.dequeue()
		if err != nil {
			return err
		}

		s.log("dequeued %q, queue size is now %d", a, len(s.queue))

		go func() {
			select {
			case <-time.After(s.dequeuedTimeout):
				s.log("timed out sending to dequeued channel")
			case s.dequeued <- a:
			}
		}()

		encoding := s.encoding
		if a.OverrideEncoding != "" {
			s.log(fmt.Sprintf("overriding encoding, using %s instead of %s", encoding, a.OverrideEncoding))
			encoding = a.OverrideEncoding
		}

		args := []string{"-y", "-re", "-i", "pipe:", "-c:a", encoding}
		args = append(args, bitrateArgs...)
		args = append(args, "-hls_time", hlsTime, "-hls_list_size", hlsSize)
		args = append(args, "-hls_flags", "append_list+delete_segments+omit_endlist")
		args = append(args, "-metadata", "title="+s.setMetadata(a))
		args = append(args, "-master_pl_name", s.masterPlaylistName)
		args = append(args, "-hls_segment_filename", s.segmentFilename, s.playlistName)

		s.log("executing ffmpeg with args: %q", args)

		cmd := exec.Command("ffmpeg", args...)
		cmd.Stdin = a.Data
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to processes audio %s in ffmpeg: %s: %w", a, string(out), err)
		}

		s.log("ffmpeg output: %s", string(out))
	}
}

// converts bitrates into ffmpeg relevant arguments
func bitratesToArgs(rates []string) []string {
	var streamMap string
	inputs := []string{}
	for i, r := range rates {
		inputs = append(inputs, fmt.Sprintf("-b:a:%d", i), r, "-map", "a:0")
		streamMap += fmt.Sprintf(" a:%d", i)
	}

	return append(inputs, "-var_stream_map", strings.TrimSpace(streamMap))
}

func (s *Stream) dequeue() (Audio, error) {
	s.audioMux.Lock()
	defer s.audioMux.Unlock()
	if len(s.queue) == 0 {
		return Audio{}, ErrNoAudioInQueue
	}

	a := s.queue[0]
	s.queue = s.queue[1:]
	return a, nil
}

func (s Stream) log(str string, v ...interface{}) {
	if s.isLogging {
		log.Println(fmt.Sprintf("hls-audio: %s", fmt.Sprintf(str, v...)))
	}
}

func defaultMetadataTitle(a Audio) string {
	return a.Artist + " - " + a.Title
}
