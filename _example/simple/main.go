package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/davidae/go-hls-audio"
)

func main() {
	s := hls.NewStream([]string{"128k", "64k"}, hls.WithDebug())
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
