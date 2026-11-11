package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeTodoist plays Todoist's three OAuth endpoints.
type fakeTodoist struct {
	t            *testing.T
	mu           sync.Mutex
	registered   map[string]any
	registers    int
	challenge    string
	issued       int
	refreshes    int
	noRefresh    bool
	expiresIn    int
	refreshFails bool
}

func (f *fakeTodoist) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /oauth/register", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.registers++
		_ = json.NewDecoder(r.Body).Decode(&f.registered)
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"client_id":"tdd_test","token_endpoint_auth_method":"none"}`)
	})
	mux.HandleFunc("POST /oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		_ = r.ParseForm()
		if r.Form.Get("client_id") != "tdd_test" || r.Form.Has("client_secret") {
			http.Error(w, `{"error":"incorrect_application_credentials"}`, http.StatusUnauthorized)
			return
		}
		switch r.Form.Get("grant_type") {
		case "authorization_code":
			sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if r.Form.Get("code") != "the-code" || base64.RawURLEncoding.EncodeToString(sum[:]) != f.challenge {
				http.Error(w, `{"error":"bad_authorization_code"}`, http.StatusBadRequest)
				return
			}
		case "refresh_token":
			if f.refreshFails {
				http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
				return
			}
			f.refreshes++
		default:
			http.Error(w, `{"error":"unsupported_grant_type"}`, http.StatusBadRequest)
			return
		}
		f.issued++
		reply := map[string]any{
			"access_token": fmt.Sprintf("access-%d", f.issued),
			"token_type":   "Bearer",
			"expires_in":   f.expiresIn,
			"scope":        Scope,
		}
		if !f.noRefresh {
			reply["refresh_token"] = fmt.Sprintf("refresh-%d", f.issued)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(reply)
	})
	return mux
}

func setup(t *testing.T, f *fakeTodoist) *Auth {
	f.t = t
	if f.expiresIn == 0 {
		f.expiresIn = 3600
	}
	server := httptest.NewServer(f.handler())
	t.Cleanup(server.Close)
	return &Auth{
		Endpoints: Endpoints{
			Register:  server.URL + "/oauth/register",
			Authorize: "https://todoist.invalid/oauth/authorize",
			Token:     server.URL + "/oauth/access_token",
		},
		Store: &Memory{},
		HTTP:  server.Client(),
	}
}

// browser plays the user approving the request: it checks the authorization
// URL and follows the redirect Todoist would send.
func browser(t *testing.T, f *fakeTodoist, mutate func(url.Values)) func(string) error {
	return func(raw string) error {
		u, err := url.Parse(raw)
		if err != nil {
			return err
		}
		q := u.Query()
		for key, want := range map[string]string{
			"client_id":             "tdd_test",
			"scope":                 Scope,
			"response_type":         "code",
			"code_challenge_method": "S256",
		} {
			if got := q.Get(key); got != want {
				t.Errorf("authorization URL %s = %q, want %q", key, got, want)
			}
		}
		redirect, err := url.Parse(q.Get("redirect_uri"))
		if err != nil || redirect.Hostname() != "127.0.0.1" || redirect.Port() == "" || redirect.Path != "/callback" {
			t.Errorf("redirect_uri %q", q.Get("redirect_uri"))
		}
		f.mu.Lock()
		f.challenge = q.Get("code_challenge")
		f.mu.Unlock()

		reply := url.Values{"code": {"the-code"}, "state": {q.Get("state")}}
		if mutate != nil {
			mutate(reply)
		}
		go func() {
			resp, err := http.Get(redirect.String() + "?" + reply.Encode())
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}
}

func login(t *testing.T, a *Auth, open func(string) error) (Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return a.Login(ctx, open, io.Discard)
}

func TestLoginRegistersOnceAndStoresTheTokens(t *testing.T) {
	f := &fakeTodoist{}
	a := setup(t, f)

	result, err := login(t, a, browser(t, f, nil))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Registered || !result.Refreshable {
		t.Errorf("result %+v", result)
	}
	if f.registered["token_endpoint_auth_method"] != "none" || f.registered["client_name"] != ClientName {
		t.Errorf("registered %v", f.registered)
	}
	if got := fmt.Sprint(f.registered["redirect_uris"]); got != "[http://127.0.0.1/callback]" {
		t.Errorf("registered redirect_uris %s", got)
	}

	result, err = login(t, a, browser(t, f, nil))
	if err != nil {
		t.Fatal(err)
	}
	if result.Registered || f.registers != 1 {
		t.Errorf("a second login registered again (%d registrations)", f.registers)
	}

	token, err := tokenFrom(t, a)
	if err != nil || token != "access-2" {
		t.Errorf("token %q, %v", token, err)
	}
}

func TestLoginRefusesAForeignState(t *testing.T) {
	f := &fakeTodoist{}
	a := setup(t, f)
	_, err := login(t, a, browser(t, f, func(v url.Values) { v.Set("state", "forged") }))
	if err == nil || !strings.Contains(err.Error(), "did not come from this login") {
		t.Fatalf("error %v", err)
	}
	if _, err := a.Store.Get(keyToken); !errors.Is(err, ErrNotFound) {
		t.Error("a token was stored")
	}
}

func TestLoginReportsADenial(t *testing.T) {
	f := &fakeTodoist{}
	a := setup(t, f)
	_, err := login(t, a, browser(t, f, func(v url.Values) { v.Del("code"); v.Set("error", "access_denied") }))
	if err == nil || !strings.Contains(err.Error(), "access_denied") {
		t.Fatalf("error %v", err)
	}
}

func TestLoginSaysWhenThereIsNoRefreshToken(t *testing.T) {
	f := &fakeTodoist{noRefresh: true}
	result, err := login(t, setup(t, f), browser(t, f, nil))
	if err != nil {
		t.Fatal(err)
	}
	if result.Refreshable {
		t.Error("Refreshable without a refresh token")
	}
}

func TestAnExpiredTokenIsRefreshedAndStored(t *testing.T) {
	// Expiring at once, so the first use refreshes.
	f := &fakeTodoist{expiresIn: 1}
	a := setup(t, f)
	if _, err := login(t, a, browser(t, f, nil)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)

	token, err := tokenFrom(t, a)
	if err != nil {
		t.Fatal(err)
	}
	if token != "access-2" || f.refreshes != 1 {
		t.Errorf("token %q after %d refreshes", token, f.refreshes)
	}
	stored, _ := a.Store.Get(keyToken)
	if !strings.Contains(stored, `"access-2"`) || !strings.Contains(stored, `"refresh-2"`) {
		t.Errorf("stored %s", stored)
	}
}

func TestALapsedSignInSaysToLogIn(t *testing.T) {
	f := &fakeTodoist{expiresIn: 1, refreshFails: true}
	a := setup(t, f)
	if _, err := login(t, a, browser(t, f, nil)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	if _, err := tokenFrom(t, a); err == nil || !strings.Contains(err.Error(), "harold login") {
		t.Fatalf("error %v", err)
	}
}

func TestSignedOut(t *testing.T) {
	a := setup(t, &fakeTodoist{})
	if _, err := a.TokenSource(context.Background()); !errors.Is(err, ErrSignedOut) {
		t.Fatalf("error %v", err)
	}
}

func TestLogoutKeepsTheRegistration(t *testing.T) {
	f := &fakeTodoist{}
	a := setup(t, f)
	if _, err := login(t, a, browser(t, f, nil)); err != nil {
		t.Fatal(err)
	}
	if err := a.Logout(); err != nil {
		t.Fatal(err)
	}
	if _, err := a.TokenSource(context.Background()); !errors.Is(err, ErrSignedOut) {
		t.Errorf("signed in after logout: %v", err)
	}
	if client, _ := a.storedClient(); client.ID != "tdd_test" {
		t.Errorf("client %+v", client)
	}
}

// A client registered under an older name, or stored before registrations
// recorded their name, is registered again at the next login, so the consent
// screen shows the name Harold has now. Until then its token still refreshes
// through it.
func TestARenamedClientIsRegisteredAgainAtLogin(t *testing.T) {
	for name, stored := range map[string]string{
		"an older name": `{"id": "tdd_old", "name": "Harold for Todoist"}`,
		"a bare old ID": "tdd_old",
	} {
		t.Run(name, func(t *testing.T) {
			f := &fakeTodoist{}
			a := setup(t, f)
			_ = a.Store.Set(keyClient, stored)
			_ = a.Store.Set(keyToken, `{"access_token": "old", "refresh_token": "r", "expiry": "2999-01-01T00:00:00Z"}`)

			source, err := a.TokenSource(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if token, _ := source.Token(); token.AccessToken != "old" {
				t.Errorf("before login, token %q", token.AccessToken)
			}

			result, err := login(t, a, browser(t, f, nil))
			if err != nil {
				t.Fatal(err)
			}
			if !result.Registered || f.registers != 1 || f.registered["client_name"] != ClientName {
				t.Errorf("registered %v times as %v", f.registers, f.registered["client_name"])
			}
			if client, _ := a.storedClient(); client.ID != "tdd_test" || client.Name != ClientName {
				t.Errorf("stored client %+v", client)
			}
		})
	}
}

func tokenFrom(t *testing.T, a *Auth) (string, error) {
	t.Helper()
	source, err := a.TokenSource(context.Background())
	if err != nil {
		return "", err
	}
	token, err := source.Token()
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}
