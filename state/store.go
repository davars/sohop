// Package state manages the state needed to implement OAuth flows and to persist session values in encrypted cookies.
package state

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/davars/sohop/globals"
	"github.com/davars/timebox"
	"github.com/golang/protobuf/ptypes"
)

//go:generate protoc state.proto --go_out=.

const (
	sessionAge           = 24 * time.Hour
	stateAge             = 5 * time.Minute
	maxRedirectURLLength = 2000
)

type contextKey int

const (
	sessionKey contextKey = iota
)

func (c *cookieStore) GetSession(req *http.Request) (session *Session) {
	if session, ok := req.Context().Value(sessionKey).(*Session); ok {
		return session
	}

	defer func() {
		// ensure we never return nil
		if session == nil {
			session = &Session{}
		}
		// ensure we only attempt to get the session once per context
		*req = *req.WithContext(context.WithValue(req.Context(), sessionKey, session))
	}()

	cookie, err := req.Cookie(c.name)
	if err != nil {
		return
	}

	session = &Session{}
	if !c.boxer.Open(cookie.Value, session) {
		session = nil
	}

	return
}

func (c *cookieStore) setCookie(rw http.ResponseWriter, name, value string, maxAge time.Duration) {
	cookie := &http.Cookie{
		Name:     name,
		Domain:   c.domain,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   int(maxAge / time.Second),
		Value:    value,
	}
	if maxAge < 0 {
		cookie.Expires = time.Unix(0, 0)
	} else if maxAge > 0 {
		cookie.Expires = globals.Clock.Now().Add(maxAge)
	}
	http.SetCookie(rw, cookie)
}

func (c *cookieStore) Authorize(rw http.ResponseWriter, req *http.Request, user string) error {
	expires := globals.Clock.Now().Add(sessionAge)
	expiresP, _ := ptypes.TimestampProto(expires) // TODO: fix before 9999-12-31
	session := &Session{User: user, Authorized: true, ExpiresAt: expiresP}
	value, err := c.boxer.Seal(session, sessionAge)
	if err != nil {
		return err
	}
	c.setCookie(rw, c.name, value, sessionAge)
	return nil
}

func (c *cookieStore) IsAuthorized(req *http.Request) bool {
	return c.GetSession(req).Authorized
}

// stateKeyLen is used to split the state into the portion used for the state param in the oauth flow, and the remainder
// set in the state cookie's value.  Use the length of the encoded nonce.
var stateKeyLen = base64.RawURLEncoding.EncodedLen(24)

func (c *cookieStore) CreateState(rw http.ResponseWriter, redirectURL string) (string, error) {
	if len(redirectURL) > maxRedirectURLLength {
		return "", fmt.Errorf("redirectURL %s... is too long", redirectURL[:maxRedirectURLLength])
	}

	state, err := c.boxer.Seal(&OAuthState{RedirectUrl: redirectURL}, stateAge)
	if err != nil {
		return "", err
	}
	stateKey := state[:stateKeyLen]
	c.setCookie(rw, stateKey, state[stateKeyLen:], stateAge)
	return stateKey, nil
}

func (c *cookieStore) RedeemState(rw http.ResponseWriter, req *http.Request, stateKey string) (string, error) {
	cookie, err := req.Cookie(stateKey)
	if err != nil {
		return "", err
	}
	os := &OAuthState{}
	if !c.boxer.Open(stateKey+cookie.Value, os) {
		return "", fmt.Errorf("invalid state")
	}
	c.setCookie(rw, stateKey, "", -1)
	return os.RedirectUrl, nil
}

type cookieStore struct {
	name   string
	domain string
	boxer  *timebox.Boxer
}

type Store interface {
	Authorize(http.ResponseWriter, *http.Request, string) error
	IsAuthorized(*http.Request) bool
	CreateState(http.ResponseWriter, string) (string, error)
	RedeemState(http.ResponseWriter, *http.Request, string) (string, error)
	GetSession(*http.Request) *Session
}

// New returns a new cookieStore to manage the oauth state and user sessions using encrypted cookies
func New(name, secret, domain string) (Store, error) {
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	boxer, err := timebox.New(secret)
	if err != nil {
		return nil, err
	}
	return &cookieStore{
		name:   name,
		domain: domain,
		boxer:  boxer,
	}, nil
}
