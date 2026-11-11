// Package auth signs Harold in to Todoist and keeps it signed in.
//
// OAuth 2.0 for a native app (RFC 8252): Harold registers itself as a public
// client, then runs the authorization code flow with PKCE through the system
// browser and a loopback redirect. The client ID and tokens live in a Store,
// the OS keyring in use. docs/design.md, Signing in, has what was measured.
package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// Scope is what Harold asks for: read-only, so Todoist itself refuses a write
// (AGENTS.md, Rules that protect the todo list).
const Scope = "data:read"

// ClientName is shown on Todoist's consent screen. Todoist's brand rules
// forbid Todoist as the primary name, and Harold's name does not mention it
// (AGENTS.md).
const ClientName = "Harold"

// ClientURI is the repository, linked from the consent screen; its README
// carries the disclaimer the brand rules require.
const ClientURI = "https://github.com/DArtagan/harold"

// redirectURI is registered without a port; Todoist accepts it on any port.
const redirectURI = "http://127.0.0.1/callback"

// Endpoints are where the flow goes; Todoist's in use, a test server's in
// tests.
type Endpoints struct {
	Register  string
	Authorize string
	Token     string
}

// Todoist is Todoist's OAuth endpoints.
var Todoist = Endpoints{
	Register:  "https://api.todoist.com/oauth/register",
	Authorize: "https://app.todoist.com/oauth/authorize",
	Token:     "https://api.todoist.com/oauth/access_token",
}

// ErrSignedOut means there is no token: `harold login` is needed.
var ErrSignedOut = errors.New("not signed in to Todoist: run `harold login`")

// Keys in the Store.
const (
	keyClient = "client"
	keyToken  = "token"
)

// Auth signs in against one set of endpoints and keeps its state in a Store.
type Auth struct {
	Endpoints Endpoints
	Store     Store
	// HTTP is used for registration and token requests.
	HTTP *http.Client
}

// New returns an Auth against Todoist, keeping its state in store.
func New(store Store) *Auth {
	return &Auth{Endpoints: Todoist, Store: store, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

// Result says what a login obtained.
type Result struct {
	// Refreshable is false when Todoist issued no refresh token, so the token
	// will lapse and need another login.
	Refreshable bool
	Expiry      time.Time
	// Registered is true when this login registered a new client.
	Registered bool
}

// Login runs the browser flow and stores the tokens. open is given the URL to
// visit; whatever it does, the URL is also written to out, for a machine with
// no browser.
func (a *Auth) Login(ctx context.Context, open func(url string) error, out io.Writer) (Result, error) {
	clientID, registered, err := a.client(ctx)
	if err != nil {
		return Result{}, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return Result{}, fmt.Errorf("listening for Todoist's redirect: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	config := a.config(clientID, fmt.Sprintf("http://127.0.0.1:%d/callback", port))

	state, err := random()
	if err != nil {
		return Result{}, err
	}
	verifier := oauth2.GenerateVerifier()
	authURL := config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))

	codes := make(chan string, 1)
	failures := make(chan error, 1)
	var once sync.Once
	server := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}
			query := r.URL.Query()
			code, problem := query.Get("code"), ""
			switch {
			case query.Get("state") != state:
				problem = "The sign-in did not come from this login. Run harold login again."
			case query.Get("error") != "":
				problem = "Todoist said: " + query.Get("error")
			case code == "":
				problem = "Todoist sent no authorization code."
			}
			once.Do(func() {
				if problem != "" {
					failures <- errors.New(problem)
				} else {
					codes <- code
				}
			})
			page(w, problem)
		}),
	}
	go func() { _ = server.Serve(listener) }()
	defer func() { _ = server.Close() }()

	_, _ = fmt.Fprintf(out, "Opening Todoist to sign in. If no browser opens, visit:\n\n  %s\n\n", authURL)
	if err := open(authURL); err != nil {
		_, _ = fmt.Fprintf(out, "(No browser: %v)\n", err)
	}

	var code string
	select {
	case code = <-codes:
	case err := <-failures:
		return Result{}, err
	case <-ctx.Done():
		return Result{}, fmt.Errorf("waiting for the sign-in: %w", ctx.Err())
	}

	token, err := config.Exchange(a.context(ctx), code, oauth2.VerifierOption(verifier))
	if err != nil {
		return Result{}, fmt.Errorf("exchanging the authorization code: %w", err)
	}
	if err := a.saveToken(token); err != nil {
		return Result{}, err
	}
	return Result{Refreshable: token.RefreshToken != "", Expiry: token.Expiry, Registered: registered}, nil
}

// Logout forgets the tokens. The client registration is kept, so the next
// login does not register again.
func (a *Auth) Logout() error {
	return a.Store.Delete(keyToken)
}

// TokenSource returns tokens for API calls, refreshing an expired one and
// storing what the refresh returns.
func (a *Auth) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	// The client the token was issued to, whatever its name now: a token is
	// refreshed by the client that holds it.
	client, err := a.storedClient()
	if errors.Is(err, ErrNotFound) {
		return nil, ErrSignedOut
	}
	if err != nil {
		return nil, err
	}
	clientID := client.ID
	stored, err := a.Store.Get(keyToken)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrSignedOut
	}
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	if err := json.Unmarshal([]byte(stored), &token); err != nil {
		return nil, fmt.Errorf("the stored token is unreadable, run `harold login`: %w", err)
	}
	config := a.config(clientID, redirectURI)
	return &savingSource{base: config.TokenSource(a.context(ctx), &token), auth: a, last: token.AccessToken}, nil
}

// savingSource stores each token its base source refreshes.
type savingSource struct {
	base oauth2.TokenSource
	auth *Auth
	mu   sync.Mutex
	last string
}

func (s *savingSource) Token() (*oauth2.Token, error) {
	token, err := s.base.Token()
	if err != nil {
		var retrieve *oauth2.RetrieveError
		if errors.As(err, &retrieve) {
			return nil, fmt.Errorf("the sign-in has lapsed, run `harold login`: %w", err)
		}
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if token.AccessToken != s.last {
		if err := s.auth.saveToken(token); err != nil {
			return nil, err
		}
		s.last = token.AccessToken
	}
	return token, nil
}

func (a *Auth) config(clientID, redirect string) *oauth2.Config {
	return &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  a.Endpoints.Authorize,
			TokenURL: a.Endpoints.Token,
			// A public client: the client ID goes in the form, and no secret.
			AuthStyle: oauth2.AuthStyleInParams,
		},
		RedirectURL: redirect,
		// Todoist separates scopes with commas; Harold asks for one.
		Scopes: []string{Scope},
	}
}

// context carries a.HTTP to the oauth2 package.
func (a *Auth) context(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, a.HTTP)
}

// registration is a client Harold registered, as the Store keeps it.
type registration struct {
	ID string `json:"id"`
	// Name is the client_name it was registered under, which Todoist shows
	// on the consent screen and fixes at registration.
	Name string `json:"name"`
}

// storedClient reads the stored registration. One stored as a bare client
// ID, before registrations recorded their name, reads with no name.
func (a *Auth) storedClient() (registration, error) {
	stored, err := a.Store.Get(keyClient)
	if err != nil {
		return registration{}, err
	}
	var client registration
	if json.Unmarshal([]byte(stored), &client) != nil || client.ID == "" {
		return registration{ID: stored}, nil
	}
	return client, nil
}

// client returns the client ID to sign in with, registering a client when
// none is stored or the stored one was registered under another name, so a
// change to ClientName reaches the consent screen at the next login.
func (a *Auth) client(ctx context.Context) (string, bool, error) {
	stored, err := a.storedClient()
	if err == nil && stored.Name == ClientName {
		return stored.ID, false, nil
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return "", false, err
	}
	id, err := a.register(ctx)
	if err != nil {
		return "", false, err
	}
	encoded, err := json.Marshal(registration{ID: id, Name: ClientName})
	if err != nil {
		return "", false, err
	}
	if err := a.Store.Set(keyClient, string(encoded)); err != nil {
		return "", false, err
	}
	return id, true, nil
}

func (a *Auth) register(ctx context.Context) (string, error) {
	body, err := json.Marshal(map[string]any{
		"client_name":                ClientName,
		"client_uri":                 ClientURI,
		"redirect_uris":              []string{redirectURI},
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
		"scope":                      Scope,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.Endpoints.Register, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("registering with Todoist: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	reply, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registering with Todoist: %s: %s", resp.Status, strings.TrimSpace(string(reply)))
	}
	var client struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal(reply, &client); err != nil || client.ClientID == "" {
		return "", fmt.Errorf("registering with Todoist: no client_id in the reply")
	}
	return client.ClientID, nil
}

func (a *Auth) saveToken(token *oauth2.Token) error {
	encoded, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return a.Store.Set(keyToken, string(encoded))
}

func random() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// page is what the browser shows after the redirect.
func page(w http.ResponseWriter, problem string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	message := "Harold is signed in to Todoist. You can close this tab."
	if problem != "" {
		w.WriteHeader(http.StatusBadRequest)
		message = problem
	}
	_, _ = fmt.Fprintf(w, "<!doctype html><title>Harold</title><p>%s</p>", html.EscapeString(message))
}
