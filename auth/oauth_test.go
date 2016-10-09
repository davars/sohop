package auth

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gorilla/sessions"
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
		authorized        bool
		expectsNextCalled bool
	}{
		"no auth": {
			authorized:        false,
			expectsNextCalled: false,
		},
		"auth": {
			authorized:        true,
			expectsNextCalled: true,
		},
	}

	auther := newMockAuther("error")

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Log(name)

			ts := newTestStore(t)
			if test.authorized {
				ts.s.Values[authorizedKey] = true
			}

			nextCalled := false
			handler := Middleware(ts, auther)(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				t.Log("next handler called")
				nextCalled = true
			}))

			resp := callHandler(t, handler, "/path?query=param")

			if test.expectsNextCalled {
				if !nextCalled {
					t.Errorf("nextCalled: got %t, want %t", nextCalled, test.expectsNextCalled)
				}
			} else {
				url := resp.Request.URL.String()
				if url != ts.s.Values[redirectURLKey].(string) {
					t.Errorf("redirectURL: got %q, want %q", ts.s.Values[redirectURLKey], url)
				}

				assertRedirectedTo(t, resp, auther.OAuthConfig().AuthCodeURL(ts.s.Values[stateKey].(string), oauth2.AccessTypeOffline))
			}
		})
	}
}

func TestHandler(t *testing.T) {
	auther := newMockAuther("")
	ts := newTestStore(t)
	ts.s.Values[stateKey] = "testing"
	ts.s.Values[redirectURLKey] = "/somewhere"
	handler := Handler(ts, auther)
	resp := callHandler(t, handler, "/foo?code=42&state=testing")
	url, err := resp.Location()
	require.NoError(t, err)
	assert.Equal(t, "/somewhere", url.Path)
}

func newMockAuther(err string) Auther {
	return &MockAuth{ClientID: "id", ClientSecret: "secret", User: "user", Err: err}
}

// a testStore implements store.Namer (which itself embeds sessions.Store), that uses the same session object for all
// operations.  Useful for writing tests for things that manipulate sessions.
type testStore struct {
	s *sessions.Session
}

func (ts *testStore) Name() string {
	return "test"
}

func (ts *testStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	if ts.s == nil {
		return ts.New(r, name)
	}
	return ts.s, nil
}

func (ts *testStore) New(r *http.Request, name string) (*sessions.Session, error) {
	ts.s = sessions.NewSession(ts, ts.Name())
	return ts.s, nil
}

func (ts *testStore) Save(r *http.Request, w http.ResponseWriter, s *sessions.Session) error {
	ts.s = s
	return nil
}

func newTestStore(t *testing.T) *testStore {
	ts := &testStore{}
	sess, err := ts.New(nil, ts.Name())
	assert.NoError(t, err)
	err = ts.Save(nil, nil, sess)
	assert.NoError(t, err)
	return ts
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
