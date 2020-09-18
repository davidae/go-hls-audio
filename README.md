# go-hls-audio
[![GoDoc](https://godoc.org/github.com/davidae/go-hls-audio?status.svg)](https://godoc.org/github.com/davidae/go-hls-audio)

A simple package to manage and produce a configurable HLS stream. This package uses `ffmpeg` to convert audio files 
into an HLS format. It will continuously append to the HLS playlists the audio that has been 
appended to the queue until it's empty. It's your responsibility to keep it not empty.


The goals of this package is to make it simpler to create a web/internet radio or similar.

## ffmpeg
This package was built for ffmpeg version `4.3`. Its not possible to execute the ffmpeg command on the usual `3.4` 
version that's currently on the APT. This is because the flag `-var_stream_map` is used. 
This was added in [92a32d0](https://github.com/FFmpeg/FFmpeg/commit/92a32d0747b089d46ae9bfea9ff79c74fdc4416f), 
after `3.4`.

## Usage 
A simple example for getting started,
```go
package main

import (
	"fmt"
	"os"

	"github.com/davidae/go-hls-audio"
)

func main() {
	s := hls.NewStream([]string{"64k", "128k"}, hls.WithDebug())
	song, err := os.Open("song.mp3")
	if err != nil {
		panic(err)
	}

	s.Append(hls.Audio{Data: song, Artist: "Foo", Title: "Bar"})

	go func() {
		for {
			a := <-s.Dequeued()
			fmt.Printf("dequeued %q, %d left in the queue\n", a, s.QueueSize())
		}
	}()

	if err := s.Start(); err != nil {
		if !errors.Is(err, hls.ErrNoAudioInQueue) {
			panic(err)
		}
	}
}
```
Checkout `_example/` on how to use this package and serve HLS streams on a file server.

