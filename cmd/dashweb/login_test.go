package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/dinapappor/go-dashcam/config"
)

const testPassword = "hunter2"

func testConfig(dir string) config.Config {
	return config.Config{
		OutputDirectory: dir,
		WebConfig:       &config.WebConfig{Listen: ":8080", Password: testPassword},
	}
}

func send(t *testing.T, handler http.Handler, method, path string, cookie *http.Cookie, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	for name, values := range header {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func postLogin(t *testing.T, handler http.Handler, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func sessionCookie(recorder *httptest.ResponseRecorder) *http.Cookie {
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session" {
			return cookie
		}
	}
	return nil
}

func login(t *testing.T, handler http.Handler) *http.Cookie {
	t.Helper()
	recorder := postLogin(t, handler, url.Values{"password": {testPassword}})
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("login status = %d, want %d:\n%s", recorder.Code, http.StatusSeeOther, recorder.Body.String())
	}
	cookie := sessionCookie(recorder)
	if cookie == nil || cookie.Value == "" {
		t.Fatalf("login set no session cookie")
	}
	return &http.Cookie{Name: cookie.Name, Value: cookie.Value}
}

func tagWith(body, element, attribute string) (string, bool) {
	rest := body
	for {
		start := strings.Index(rest, element)
		if start < 0 {
			return "", false
		}
		end := strings.Index(rest[start:], ">")
		if end < 0 {
			return "", false
		}
		tag := rest[start : start+end+1]
		if strings.Contains(tag, attribute) {
			return tag, true
		}
		rest = rest[start+end+1:]
	}
}

func TestLoginPage(t *testing.T) {
	recorder := send(t, newHandler(testConfig(t.TempDir())), http.MethodGet, "/login", nil, nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", contentType)
	}
	body := recorder.Body.String()

	form, found := tagWith(body, "<form", `action="/login"`)
	if !found {
		t.Fatalf("login page has no <form action=\"/login\">:\n%s", body)
	}
	if method, _ := attributeValue(form, "<form", "method"); !strings.EqualFold(method, "post") {
		t.Errorf("login form method = %q, want post", method)
	}
	input, found := tagWith(body, "<input", `type="password"`)
	if !found {
		t.Fatalf("login page has no password input:\n%s", body)
	}
	if name, _ := attributeValue(input, "<input", "name"); name != "password" {
		t.Errorf("password input name = %q, want %q", name, "password")
	}
	if !strings.Contains(body, `<link rel="stylesheet" href="/static/tabler.min.css">`) {
		t.Errorf("login page does not use the shared head:\n%s", body)
	}
}

func TestLoginCorrectPassword(t *testing.T) {
	recorder := postLogin(t, newHandler(testConfig(t.TempDir())), url.Values{"password": {testPassword}})

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusSeeOther)
	}
	if location := recorder.Header().Get("Location"); location != "/" {
		t.Errorf("Location = %q, want %q", location, "/")
	}
	cookie := sessionCookie(recorder)
	if cookie == nil {
		t.Fatal("no session cookie")
	}
	if len(cookie.Value) < 32 {
		t.Errorf("session token %q is too short, want at least 32 characters", cookie.Value)
	}
	if strings.Contains(cookie.Value, testPassword) {
		t.Errorf("session cookie contains the password")
	}
	if !cookie.HttpOnly {
		t.Errorf("session cookie is not HttpOnly")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("session cookie SameSite = %v, want Strict", cookie.SameSite)
	}
	if cookie.Path != "/" {
		t.Errorf("session cookie Path = %q, want %q", cookie.Path, "/")
	}
}

func TestLoginTokensDiffer(t *testing.T) {
	handler := newHandler(testConfig(t.TempDir()))

	first := login(t, handler)
	second := login(t, handler)

	if first.Value == second.Value {
		t.Errorf("two logins got the same token %q", first.Value)
	}
	for _, cookie := range []*http.Cookie{first, second} {
		if recorder := send(t, handler, http.MethodGet, "/", cookie, nil); recorder.Code != http.StatusOK {
			t.Errorf("GET / with token %q: status = %d, want %d", cookie.Value, recorder.Code, http.StatusOK)
		}
	}
}

func TestLoginWrongPassword(t *testing.T) {
	tests := []struct {
		name string
		form url.Values
	}{
		{"wrong password", url.Values{"password": {"wrong"}}},
		{"empty password", url.Values{"password": {""}}},
		{"no password field", url.Values{}},
		{"different case", url.Values{"password": {strings.ToUpper(testPassword)}}},
		{"only the start", url.Values{"password": {testPassword[:3]}}},
		{"extra space", url.Values{"password": {testPassword + " "}}},
		{"password in another field", url.Values{"pass": {testPassword}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := postLogin(t, newHandler(testConfig(t.TempDir())), tt.form)

			if recorder.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			if cookie := sessionCookie(recorder); cookie != nil && cookie.Value != "" {
				t.Errorf("wrong password got a session cookie %q", cookie.Value)
			}
			body := recorder.Body.String()
			if !strings.Contains(body, `action="/login"`) {
				t.Errorf("reply is not the login page:\n%s", body)
			}
			if !strings.Contains(body, "Wrong password") {
				t.Errorf("reply does not say %q:\n%s", "Wrong password", body)
			}
		})
	}
}

func TestLoginWithoutConfiguredPassword(t *testing.T) {
	tests := []struct {
		name string
		web  *config.WebConfig
	}{
		{"no web config", nil},
		{"empty password", &config.WebConfig{Listen: ":8080"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newHandler(config.Config{OutputDirectory: t.TempDir(), WebConfig: tt.web})

			recorder := postLogin(t, handler, url.Values{"password": {""}})

			if recorder.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			if cookie := sessionCookie(recorder); cookie != nil && cookie.Value != "" {
				t.Errorf("got a session cookie %q", cookie.Value)
			}
		})
	}
}

var protectedRoutes = []struct {
	method string
	path   string
}{
	{http.MethodGet, "/"},
	{http.MethodGet, "/recordings"},
	{http.MethodGet, "/play/" + finishedRecording},
	{http.MethodGet, "/video/" + finishedRecording},
	{http.MethodGet, "/video/" + finishedRecording + "?download=1"},
	{http.MethodDelete, "/video/" + finishedRecording},
}

func TestProtectedRoutesRedirectToLogin(t *testing.T) {
	cookies := []struct {
		name   string
		cookie *http.Cookie
	}{
		{"no cookie", nil},
		{"unknown token", &http.Cookie{Name: "session", Value: "0123456789abcdef0123456789abcdef"}},
		{"empty token", &http.Cookie{Name: "session", Value: ""}},
	}

	for _, route := range protectedRoutes {
		for _, c := range cookies {
			t.Run(route.method+" "+route.path+" "+c.name, func(t *testing.T) {
				setup := newDeleteSetup(t)

				recorder := send(t, setup.handler, route.method, route.path, c.cookie, nil)

				if recorder.Code != http.StatusSeeOther {
					t.Errorf("status = %d, want %d", recorder.Code, http.StatusSeeOther)
				}
				if location := recorder.Header().Get("Location"); location != "/login" {
					t.Errorf("Location = %q, want %q", location, "/login")
				}
				if strings.Contains(recorder.Body.String(), finishedRecording) || recorder.Body.String() == string(finishedContent) {
					t.Errorf("reply leaks recording data:\n%s", recorder.Body.String())
				}
				assertNothingDeleted(t, setup)
			})
		}
	}
}

func TestProtectedRoutesHTMX(t *testing.T) {
	for _, route := range protectedRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			setup := newDeleteSetup(t)

			recorder := send(t, setup.handler, route.method, route.path, nil, http.Header{"Hx-Request": {"true"}})

			if recorder.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			if redirect := recorder.Header().Get("HX-Redirect"); redirect != "/login" {
				t.Errorf("HX-Redirect = %q, want %q", redirect, "/login")
			}
			if strings.Contains(recorder.Body.String(), finishedRecording) {
				t.Errorf("reply leaks recording data:\n%s", recorder.Body.String())
			}
			assertNothingDeleted(t, setup)
		})
	}
}

func TestPublicRoutes(t *testing.T) {
	handler := newHandler(testConfig(t.TempDir()))
	for _, path := range []string{
		"/login",
		"/static/htmx.min.js",
		"/static/tabler.min.css",
	} {
		t.Run(path, func(t *testing.T) {
			recorder := send(t, handler, http.MethodGet, path, nil, nil)
			if recorder.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", recorder.Code, http.StatusOK)
			}
		})
	}
}

func TestSessionsBelongToOneServer(t *testing.T) {
	dir := t.TempDir()
	first := newHandler(testConfig(dir))
	second := newHandler(testConfig(dir))

	cookie := login(t, first)
	recorder := send(t, second, http.MethodGet, "/", cookie, nil)

	if recorder.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d: a restarted server must not accept old sessions", recorder.Code, http.StatusSeeOther)
	}
}

func TestLogout(t *testing.T) {
	handler := newHandler(testConfig(t.TempDir()))
	cookie := login(t, handler)
	other := login(t, handler)

	recorder := send(t, handler, http.MethodPost, "/logout", cookie, nil)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusSeeOther)
	}
	if location := recorder.Header().Get("Location"); location != "/login" {
		t.Errorf("Location = %q, want %q", location, "/login")
	}
	cleared := sessionCookie(recorder)
	if cleared == nil {
		t.Errorf("logout does not clear the session cookie")
	} else if cleared.Value != "" || cleared.MaxAge >= 0 {
		t.Errorf("cleared cookie Value = %q MaxAge = %d, want empty value and negative MaxAge", cleared.Value, cleared.MaxAge)
	}

	if recorder := send(t, handler, http.MethodGet, "/", cookie, nil); recorder.Code != http.StatusSeeOther {
		t.Errorf("old session after logout: status = %d, want %d", recorder.Code, http.StatusSeeOther)
	}
	if recorder := send(t, handler, http.MethodGet, "/", other, nil); recorder.Code != http.StatusOK {
		t.Errorf("other session after logout: status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestLogoutWithoutSession(t *testing.T) {
	recorder := send(t, newHandler(testConfig(t.TempDir())), http.MethodPost, "/logout", nil, nil)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusSeeOther)
	}
	if location := recorder.Header().Get("Location"); location != "/login" {
		t.Errorf("Location = %q, want %q", location, "/login")
	}
}

func TestNavbarLogoutButton(t *testing.T) {
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
			form, found := tagWith(nav, "<form", `action="/logout"`)
			if !found {
				t.Fatalf("<nav> has no <form action=\"/logout\">:\n%s", nav)
			}
			if method, _ := attributeValue(form, "<form", "method"); !strings.EqualFold(method, "post") {
				t.Errorf("logout form method = %q, want post", method)
			}
			if !strings.Contains(nav, "Log out") {
				t.Errorf("<nav> has no Log out text:\n%s", nav)
			}
		})
	}
}
