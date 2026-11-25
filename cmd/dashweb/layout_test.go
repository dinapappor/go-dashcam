package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/dinapappor/go-dashcam/config"
)

func TestStaticTabler(t *testing.T) {
	recorder := get(t, newHandler(config.Config{OutputDirectory: t.TempDir()}), "/static/tabler.min.css")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/css") {
		t.Errorf("Content-Type = %q, want it to contain %q", contentType, "text/css")
	}
	if !strings.Contains(strings.ToLower(recorder.Body.String()), "tabler") {
		t.Errorf("body does not look like the Tabler stylesheet")
	}
}

func TestFullPagesHaveSharedHead(t *testing.T) {
	handler := playHandler(t)
	for _, path := range []string{
		"/",
		"/play/" + finishedRecording,
	} {
		t.Run(path, func(t *testing.T) {
			recorder := get(t, handler, path)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
			}
			body := recorder.Body.String()
			for _, want := range []string{
				`<meta name="viewport" content="width=device-width, initial-scale=1">`,
				`<link rel="stylesheet" href="/static/tabler.min.css">`,
				`<script src="/static/htmx.min.js"></script>`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("page does not contain %q:\n%s", want, body)
				}
			}
		})
	}
}

func TestPlayVideoFullWidth(t *testing.T) {
	recorder := get(t, playHandler(t), "/play/"+finishedRecording)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	class, found := attributeValue(recorder.Body.String(), "<video", "class")
	if !found {
		t.Fatalf("<video> has no class attribute:\n%s", recorder.Body.String())
	}
	if !strings.Contains(" "+class+" ", " w-100 ") {
		t.Errorf("<video> class = %q, want it to include %q", class, "w-100")
	}
}

func navSection(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, "<nav")
	if start < 0 {
		t.Fatalf("page has no <nav>:\n%s", body)
	}
	end := strings.Index(body[start:], "</nav>")
	if end < 0 {
		t.Fatalf("<nav> is not closed:\n%s", body)
	}
	return body[start : start+end]
}

func linkTo(section, text string) (string, bool) {
	at := strings.Index(section, text)
	if at < 0 {
		return "", false
	}
	start := strings.LastIndex(section[:at], "<a")
	if start < 0 {
		return "", false
	}
	return attributeValue(section[start:], "<a", "href")
}

func TestFullPagesHaveNavbar(t *testing.T) {
	handler := playHandler(t)
	for _, path := range []string{
		"/",
		"/play/" + finishedRecording,
	} {
		t.Run(path, func(t *testing.T) {
			recorder := get(t, handler, path)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
			}
			nav := navSection(t, recorder.Body.String())
			href, found := linkTo(nav, "Recordings")
			if !found {
				t.Fatalf("<nav> has no Recordings link:\n%s", nav)
			}
			if href != "/" {
				t.Errorf("Recordings link href = %q, want %q", href, "/")
			}
			if strings.Contains(nav, "Settings") {
				t.Errorf("<nav> links Settings before the page exists:\n%s", nav)
			}
		})
	}
}

func TestPlayPageBackLink(t *testing.T) {
	recorder := get(t, playHandler(t), "/play/"+finishedRecording)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	href, found := linkTo(recorder.Body.String(), "Back")
	if !found {
		t.Fatalf("play page has no Back link:\n%s", recorder.Body.String())
	}
	if href != "/" {
		t.Errorf("Back link href = %q, want %q", href, "/")
	}
}
