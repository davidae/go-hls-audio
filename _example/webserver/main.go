package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidae/go-hls-audio"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
)

func main() {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	rootDir := "/assets"
	wd, _ := os.Getwd()
	FileServer(r, rootDir, http.Dir(filepath.Join(wd, "assets")))

	go func() {
		panic(startHLS(rootDir))
	}()

	port := ":8080"

	fmt.Printf("starting file server on %s\n", port)
	fmt.Printf("try 'mplayer http://localhost%s/assets/hls/master.m3u8' now\n", port)
	http.ListenAndServe(port, r)
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

func startHLS(rootDir string) error {
	s := hls.NewStream(
		[]string{"128k", "64k"},
		hls.WithMasterPlaylistName("master.m3u8"),
		hls.WithPlaylistName(filepath.Join(rootDir, "hls/stream-%v/playlist.m3u8")),
		hls.WithSegmentFilename(filepath.Join(rootDir, "hls/stream-%v/segment-%06d.ts")),
		hls.WithDebug(),
	)
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

	return s.Start()
}
