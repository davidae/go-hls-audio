package hls_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/davidae/go-hls-audio"
)

func TestStartWithMp3File(t *testing.T) {
	s, err := hls.NewStream([]string{"128k", "64k"},
		hls.WithSegmentFilename("_test/test-%v/segment-%d.ts"),
		hls.WithPlaylistName("_test/test-%v/playlist.m3u8"),
		hls.WithMasterPlaylistName("test.m3u8"),
	)
	if err != nil {
		t.Errorf("failed to initialize stream for test: %s", err)
	}

	song, err := os.Open("_stubs/536109__eminyildirim__water-bubble.mp3")
	if err != nil {
		t.Errorf("failed to open song used for testing: %s", err)
		return
	}

	s.Append(hls.Audio{ID: 3, Data: song, Artist: "Foo", Title: "Bar"})

	var dequeuedCount int
	go func() {
		a := <-s.Dequeued()
		dequeuedCount++
		if a.Artist != "Foo" && a.Title != "Bar" && a.ID != 3 {
			t.Errorf("unexpected audio dequeued: %v", a)
		}
	}()

	defer cleanup()

	err = s.Start()
	if !errors.Is(err, hls.ErrNoAudioInQueue) {
		t.Errorf("expected error to be ErrNoAudioInQueue, but got %s", err)
		return
	}

	if dequeuedCount != 1 {
		t.Errorf("expected %d to have been dequeued, but got %d", 1, dequeuedCount)
		return
	}

	out, err := ioutil.ReadFile("_test/test.m3u8")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	const expectedMaster = "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-STREAM-INF:BANDWIDTH=140800,CODECS=\"mp4a.40.2\"\ntest-0/playlist.m3u8\n\n#EXT-X-STREAM-INF:BANDWIDTH=70400,CODECS=\"mp4a.40.2\"\ntest-1/playlist.m3u8\n\n"

	if string(out) != expectedMaster {
		t.Errorf("expected to get %q, but got %q", expectedMaster, string(out))
	}

	if err := isFilesInDirectory("_test/test-0",
		"playlist.m3u8",
		"segment-0.ts",
		"segment-1.ts",
	); err != nil {
		t.Error(err)
		return
	}

	if err := isFilesInDirectory("_test/test-1",
		"playlist.m3u8",
		"segment-0.ts",
		"segment-1.ts",
	); err != nil {
		t.Error(err)
		return
	}

	const expectedPlaylist = "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:5\n#EXT-X-MEDIA-SEQUENCE:0\n#EXT-X-DISCONTINUITY\n#EXTINF:5.015511,\nsegment-0.ts\n#EXTINF:1.696289,\nsegment-1.ts\n"

	out, err = ioutil.ReadFile("_test/test-0/playlist.m3u8")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	if string(out) != expectedPlaylist {
		t.Errorf("expected to get %q, but got %q", expectedPlaylist, string(out))
		return
	}

	out, err = ioutil.ReadFile("_test/test-1/playlist.m3u8")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	if string(out) != expectedPlaylist {
		t.Errorf("expected to get %q, but got %q", expectedPlaylist, string(out))
		return
	}
}

func TestNewStreamErrorWhenMasterPlaylistPointToDirectory(t *testing.T) {
	const expectedErr = "master playlist cannot have a directory, it must be root. ffmpeg limitation"
	_, err := hls.NewStream([]string{"128k", "64k"},
		hls.WithMasterPlaylistName("hello/master.m3u8"),
		hls.WithDebug())
	if err == nil {
		t.Errorf("expected err")
	}
	if expectedErr != err.Error() {
		t.Errorf("expected error to be %q, but got %q", expectedErr, err)
	}
}

func isFilesInDirectory(path string, files ...string) error {
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf("unexpected error: %s", err)
	}

	if len(infos) != len(files) {
		return fmt.Errorf("expected %d files in the directory, but got %d", len(files), len(infos))
	}

	m := make(map[string]struct{})
	for _, i := range infos {
		m[i.Name()] = struct{}{}
	}

	for _, f := range files {
		if _, ok := m[f]; !ok {
			return fmt.Errorf("unable to find file %s in directory: %s", f, m)
		}
	}

	return nil
}

func cleanup() {
	if err := os.RemoveAll("_test"); err != nil {
		panic(fmt.Errorf("failed to clean up after tests: %s", err))
	}
}
