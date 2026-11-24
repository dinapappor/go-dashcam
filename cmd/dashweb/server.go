package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/dinapappor/go-dashcam/config"
)

//go:embed templates
var templateFiles embed.FS

//go:embed static
var staticFiles embed.FS

var pages = template.Must(
	template.New("").
		Funcs(template.FuncMap{"formatSize": formatSize}).
		ParseFS(templateFiles, "templates/*.html"),
)

func recordingStatus(dir, name string) int {
	if !strings.HasPrefix(name, "dashcam_") || !strings.HasSuffix(name, ".mp4") {
		return http.StatusNotFound
	}
	if strings.Contains(name, "/") {
		return http.StatusNotFound
	}
	recordings, err := listRecordingInfo(dir)

	if err != nil {
		return http.StatusNotFound
	}

	for _, recording := range recordings {
		if recording.Name == name {
			if recording.Active {
				return http.StatusConflict
			} else {
				return http.StatusOK
			}
		}
	}

	return http.StatusNotFound
}

func newHandler(cfg config.Config) http.Handler {

	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.FileServerFS(staticFiles))

	mux.HandleFunc("GET /play/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		status := recordingStatus(cfg.OutputDirectory, name)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		err := pages.ExecuteTemplate(w, "play.html", name)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Unable to execute template: %v", err)
			return
		}
	})

	mux.HandleFunc("GET /video/{name}", func(w http.ResponseWriter, r *http.Request) {

		name := r.PathValue("name")

		status := recordingStatus(cfg.OutputDirectory, name)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		w.Header().Set("Content-Type", "video/mp4")
		if r.URL.Query().Get("download") == "1" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))
		}
		http.ServeFile(w, r, filepath.Join(cfg.OutputDirectory, name))
	})

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		err := pages.ExecuteTemplate(w, "index.html", nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Could not Execute template: %v", err)
			return
		}
	})

	mux.HandleFunc("GET /recordings", func(w http.ResponseWriter, r *http.Request) {
		recordings, err := listRecordingInfo(cfg.OutputDirectory)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, "No recordings")
			return
		}

		if len(recordings) == 0 {
			io.WriteString(w, "No recordings")
			return
		}
		w.Header().Set("Content-Type", "text/html")
		err = pages.ExecuteTemplate(w, "recordings.html", recordings)

		if err != nil {
			log.Printf("Could not render the recordings: %v", err)
			return
		}

	})
	return mux

}

func serve(ctx context.Context, listener net.Listener, handler http.Handler) error {
	server := &http.Server{Handler: handler}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	err := <-serveErr
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func main() {
	configFile := flag.String("config", "dashcam_config.json", "Config file to use")
	flag.Parse()

	cfg, err := config.ParseConfig(*configFile)

	if err != nil {
		log.Printf("Could not open configfile: %v", err)
		return
	}

	if cfg.WebConfig == nil {
		log.Printf("WebConfig empty, bailing")
		return
	}

	listener, err := net.Listen("tcp", cfg.WebConfig.Listen)
	if err != nil {
		log.Printf("Could not start listener: %v", err)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	err = serve(ctx, listener, newHandler(cfg))
	if err != nil {
		log.Printf("Could not start server: %v", err)
		return
	}
}
