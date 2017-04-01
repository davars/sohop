package state

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testSecret = "3c0767ada2466a92a59c1214061441713aeafe6d115e29aa376c0f9758cdf0f5"

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		domain string

		err string
	}{
		{
			err: "name cannot be empty",
		},
		{
			name: "foo",
			err:  "domain cannot be empty",
		},
		{
			name:   "foo",
			domain: "foo",
			err:    "cookie secret should be a 64-character hex-encoded string",
		},
		{
			name:   "foo",
			domain: "foo",
			secret: "not long enough or hex-encoded",
			err:    "cookie secret should be a 64-character hex-encoded string",
		},
		{
			name:   "foo",
			domain: "foo",
			secret: testSecret,
			err:    "<nil>",
		},
	}
	for _, test := range tests {
		_, err := New(test.name, test.secret, test.domain)
		if err == nil {
			err = fmt.Errorf("%v", nil)
		}
		if !strings.Contains(err.Error(), test.err) {
			t.Errorf("got %q, want something containing %q", err.Error(), test.err)
		}
	}
}

func TestCookieStore_Session(t *testing.T) {
	store, err := New("test", testSecret, "example.com")
	assert.NoError(t, err)
	req, err := http.NewRequest("GET", "http://example.com", nil)
	assert.NoError(t, err)

	userName := "testUser"

	rw := httptest.NewRecorder()
	err = store.Authorize(rw, req, userName)
	assert.NoError(t, err)

	cookieHeader := rw.HeaderMap["Set-Cookie"]
	assert.Equal(t, 1, len(cookieHeader))
	cookie := cookieHeader[0]
	for _, s := range []string{"test=", "Path=/; Domain=example.com; Expires=", "Max-Age=86400; HttpOnly; Secure"} {
		if !strings.Contains(cookie, s) {
			t.Fatalf("expected cookie to contain %q\nwas: %q", s, cookie)
		}
	}
	sealed := strings.Replace(strings.Split(cookie, ";")[0], "test=", "", 1)
	session := &Session{}
	assert.True(t, store.(*cookieStore).boxer.open(sealed, session))
	assert.True(t, session.Authorized)
	assert.Equal(t, userName, session.User)
}

func TestCookieStore_State(t *testing.T) {
	store, err := New("test", testSecret, "example.com")
	assert.NoError(t, err)

	redirectURL := "http://example.com/someplaceauthenticated"

	rw := httptest.NewRecorder()
	stateKey, err := store.CreateState(rw, redirectURL)
	assert.NoError(t, err)

	cookieHeader := rw.HeaderMap["Set-Cookie"]
	assert.Equal(t, 1, len(cookieHeader))
	cookie := cookieHeader[0]
	for _, s := range []string{stateKey + "=", "Path=/; Domain=example.com; Expires=", "Max-Age=300; HttpOnly; Secure"} {
		if !strings.Contains(cookie, s) {
			t.Fatalf("expected cookie to contain %q\nwas: %q", s, cookie)
		}
	}
	sealed := stateKey + strings.Replace(strings.Split(cookie, ";")[0], stateKey+"=", "", 1)
	oauthState := &OAuthState{}
	assert.True(t, store.(*cookieStore).boxer.open(sealed, oauthState))
	assert.Equal(t, redirectURL, oauthState.RedirectUrl)

	req, err := http.NewRequest("GET", "http://example.com", nil)
	assert.NoError(t, err)
	req.Header.Add("Cookie", cookie)
	rw = httptest.NewRecorder()
	state, err := store.RedeemState(rw, req, stateKey)
	assert.NoError(t, err)
	assert.Equal(t, redirectURL, state)
}
