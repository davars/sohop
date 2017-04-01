package auth

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/davars/sohop/state"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func TestNewAuther(t *testing.T) {
	tests := []struct {
		in  Config
		out Auther
		err string
	}{
		{
			in: Config{
				Type:   "none",
				Config: []byte{},
			},
			err: `unknown auther type "none"`,
		},
		{
			in: Config{
				Type:   "github-org",
				Config: []byte(`{"ClientID": "id", "ClientSecret": "secret", "OrgID": 42}`),
			},
			out: &GithubAuth{ClientID: "id", ClientSecret: "secret", OrgID: 42},
		},
		{
			in: Config{
				Type: "gmail-regex",
				Config: []byte(`{
				"Credentials": {"web":{"client_id":"client-id","project_id":"example","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://accounts.google.com/o/oauth2/token","auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs","client_secret":"client-secret","redirect_uris":["https://oauth.example.com/authorized"]}},
				"EmailRegex":"^test@example.com$"
				}`),
			},
			out: &GoogleAuth{
				config: &oauth2.Config{
					ClientID:     "client-id",
					ClientSecret: "client-secret",
					Endpoint:     google.Endpoint,
					RedirectURL:  "https://oauth.example.com/authorized",
					Scopes:       []string{"openid", "email"},
				},
				emailRegex: regexp.MustCompile("^test@example.com$"),
			},
		},
		{
			in: Config{
				Type:   "mock",
				Config: []byte(`{"ClientID": "id", "ClientSecret": "secret", "User": "user", "Err": "error"}`),
			},
			out: newMockAuther("error"),
		},
		{
			in: Config{
				Type:   "gmail-regex",
				Config: []byte(`{}`),
			},
			out: &GoogleAuth{},
			err: "unexpected end of JSON input",
		},
	}

	for _, test := range tests {
		auth, err := NewAuther(test.in)
		assert.Equal(t, test.out, auth)
		if test.err == "" {
			assert.NoError(t, err)
		} else {
			assert.Equal(t, test.err, err.Error())
		}
	}
}

func TestMiddleware(t *testing.T) {
	tests := map[string]struct {
		session           *state.Session
		expectsNextCalled bool
	}{
		"no auth": {
			session:           &state.Session{},
			expectsNextCalled: false,
		},
		"auth": {
			session:           &state.Session{Authorized: true},
			expectsNextCalled: true,
		},
	}

	auther := newMockAuther("error")

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Log(name)

			ts := newTestStore(t, test.session, map[string]*state.OAuthState{})
			nextCalled := false
			handler := Middleware(auther, ts)(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				t.Log("next handler called")
				nextCalled = true
			}))

			resp := callHandler(t, handler, "/path?query=param")

			if test.expectsNextCalled {
				if !nextCalled {
					t.Errorf("nextCalled: got %t, want %t", nextCalled, test.expectsNextCalled)
				}
			} else {
				loc, err := resp.Location()
				assert.NoError(t, err)

				key := loc.Query().Get("state")
				url := resp.Request.URL.String()
				redirectURL := ts.oauthStates[key].RedirectUrl
				if url != redirectURL {
					t.Errorf("redirectURL: got %q, want %q", redirectURL, url)
				}
				assertRedirectedTo(t, resp, auther.OAuthConfig().AuthCodeURL(key, oauth2.AccessTypeOffline))
			}
		})
	}
}

func TestHandler(t *testing.T) {
	auther := newMockAuther("")
	redirectURL := "https://some.other/place"

	ts := newTestStore(t, nil, map[string]*state.OAuthState{
		"testing": {RedirectUrl: redirectURL},
	})
	handler := Handler(auther, ts)
	resp := callHandler(t, handler, "/foo?code=42&state=testing")
	url, err := resp.Location()
	require.NoError(t, err)
	assert.Equal(t, redirectURL, url.String())
}

func newMockAuther(err string) Auther {
	return &MockAuth{ClientID: "id", ClientSecret: "secret", User: "user", Err: err}
}

// a testStore implements store.Namer (which itself embeds sessions.Config), that uses the same session object for all
// operations.  Useful for writing tests for things that manipulate sessions.
type testStore struct {
	session     *state.Session
	oauthStates map[string]*state.OAuthState
}

func (ts *testStore) GetSession(req *http.Request) (session *state.Session) {
	if ts.session == nil {
		ts.session = &state.Session{}
	}
	return ts.session
}

func (ts *testStore) Authorize(rw http.ResponseWriter, req *http.Request, user string) error {
	ts.session.Authorized = true
	ts.session.User = user
	return nil
}

func (ts *testStore) IsAuthorized(req *http.Request) bool {
	return ts.GetSession(req).Authorized
}

func (ts *testStore) CreateState(rw http.ResponseWriter, redirectURL string) (string, error) {
	h := md5.New()
	io.WriteString(h, redirectURL)
	key := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	ts.oauthStates[key] = &state.OAuthState{RedirectUrl: redirectURL}
	return key, nil
}

func (ts *testStore) RedeemState(rw http.ResponseWriter, req *http.Request, stateKey string) (string, error) {
	if state, ok := ts.oauthStates[stateKey]; ok {
		return state.RedirectUrl, nil
	}
	return "", fmt.Errorf("not found")
}

func newTestStore(t *testing.T, session *state.Session, oauthState map[string]*state.OAuthState) *testStore {
	return &testStore{
		session:     session,
		oauthStates: oauthState,
	}
}

// noRedirectClient return an *http.Client that never follows redirects
func noRedirectClient(t *testing.T) *http.Client {
	cli := &http.Client{}
	cli.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		t.Log("cancelling redirect")
		return http.ErrUseLastResponse
	}
	return cli
}

// callHandler invokes handler as part of an http server request (rather than simply calling ServeHTTP).  Not very
// useful outside of testing middleware.
func callHandler(t *testing.T, handler http.Handler, uri string) *http.Response {
	server := httptest.NewServer(handler)
	defer server.Close()

	url := server.URL + uri

	resp, err := noRedirectClient(t).Get(url)
	assert.NoError(t, err)

	return resp
}

func assertRedirectedTo(t *testing.T, resp *http.Response, url string) {
	redirectedTo, err := resp.Location()
	assert.NoError(t, err)
	if redirectedTo.String() != url {
		t.Errorf("assertRedirectedTo: got %q, want %q", redirectedTo.String(), url)
	}
}
