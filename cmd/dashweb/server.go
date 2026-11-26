package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
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
	"sync"
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

func writeRecordings(w http.ResponseWriter, dir string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	recordings, err := listRecordingInfo(dir)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "No recordings")
		return
	}

	err = pages.ExecuteTemplate(w, "recordings.html", recordings)
	if err != nil {
		log.Printf("Could not render the recordings: %v", err)
	}
}

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

	var sessionMutex sync.Mutex
	sessions := map[string]bool{}

	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.FileServerFS(staticFiles))

	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		err := pages.ExecuteTemplate(w, "login.html", nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Unable to execute template: %v", err)
			return
		}
	})

	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		if cfg.WebConfig == nil {
			w.WriteHeader(http.StatusUnauthorized)
			log.Printf("Password unconfigured")
			return
		}

		if len(cfg.WebConfig.Password) == 0 {
			w.WriteHeader(http.StatusUnauthorized)

			err := pages.ExecuteTemplate(w, "login.html", "Wrong password")
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Printf("Unable to execute template: %v", err)
			}
			return

		}
		configuredPassword := []byte(cfg.WebConfig.Password)

		enteredPassword := []byte(r.FormValue("password"))

		if subtle.ConstantTimeCompare(configuredPassword, enteredPassword) != 1 {
			w.WriteHeader(http.StatusUnauthorized)
			err := pages.ExecuteTemplate(w, "login.html", "Wrong password")
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Printf("Unable to execute template: %v", err)
			}
			return
		}

		sessionToken := rand.Text() + rand.Text()

		sessionMutex.Lock()
		sessions[sessionToken] = true
		sessionMutex.Unlock()

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    sessionToken,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)

	})

	mux.HandleFunc("POST /logout", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err == nil {
			sessionMutex.Lock()
			delete(sessions, cookie.Value)
			sessionMutex.Unlock()
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	mux.HandleFunc("DELETE /video/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		status := recordingStatus(cfg.OutputDirectory, name)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}

		err := os.Remove(filepath.Join(cfg.OutputDirectory, name))

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		writeRecordings(w, cfg.OutputDirectory)
	})

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
		writeRecordings(w, cfg.OutputDirectory)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" || strings.HasPrefix(r.URL.Path, "/static/") {
			mux.ServeHTTP(w, r)
			return
		}

		loggedIn := false
		cookie, err := r.Cookie("session")
		if err == nil && cookie.Value != "" {
			sessionMutex.Lock()
			loggedIn = sessions[cookie.Value]
			sessionMutex.Unlock()
		}

		if !loggedIn {
			if r.Header.Get("HX-Request") != "" {
				w.Header().Set("HX-Redirect", "/login")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		mux.ServeHTTP(w, r)
	})
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
