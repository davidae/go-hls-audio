package hls

import "time"

// StreamOption is a func used to configure the streamer upon initialization
type StreamOption func(s *Stream)

// WithBitrates sets the ffmpeg bitrate, default is 128k
func WithBitrates(bitrates []string) StreamOption {
	return func(s *Stream) { s.bitrates = bitrates }
}

// WithHLSSize sets the ffmpeg hls_list_size, default is 50
func WithHLSSize(size int) StreamOption {
	return func(s *Stream) { s.hlsListSize = size }
}

// WithHLSTime sets the ffmpeg hls_time, default is 8
func WithHLSTime(t int) StreamOption {
	return func(s *Stream) { s.hlsTime = t }
}

// WithSegmentFilename sets segment file names. It MUST use the ffmpeg filename format, so you
// must include %v and %d. %v specifies the position of variant stream index in
// the generated segment file names
func WithSegmentFilename(name string) StreamOption {
	return func(s *Stream) { s.segmentFilename = name }
}

// WithPlaylistName sets playlists names. It MUST use the ffmpeg filename format, so you
// must include %v and %d. %v specifies the position of variant stream index in
// the generated segment file names
func WithPlaylistName(name string) StreamOption {
	return func(s *Stream) { s.playlistName = name }
}

// WithMasterPlaylistName sets the master playlist name
func WithMasterPlaylistName(name string) StreamOption {
	return func(s *Stream) { s.masterPlaylistName = name }
}

// WithDequeuedTimeout sets the queue channel timeout
func WithDequeuedTimeout(d time.Duration) StreamOption {
	return func(s *Stream) { s.dequeuedTimeout = d }
}

// WithDebug allows for debug logging
func WithDebug() StreamOption {
	return func(s *Stream) { s.isLogging = true }
}

// WithMetadataTitle sets the -metadata value in ffmpeg hls, as in "-metadata title=%s"
func WithMetadataTitle(fn func(a Audio) string) StreamOption {
	return func(s *Stream) { s.setMetadata = fn }
}
