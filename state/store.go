// Package state manages the state needed to implement OAuth flows and to persist session values in encrypted cookies.
package state

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/davars/sohop/globals"
	"github.com/golang/protobuf/ptypes"
)

//go:generate protoc state.proto --go_out=.

const (
	sessionAge           = int(24 * time.Hour / time.Second)
	stateAge             = int(5 * time.Minute / time.Second)
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
	if !c.boxer.open(cookie.Value, session) {
		session = nil
	}

	return
}

func (c *cookieStore) setCookie(rw http.ResponseWriter, name, value string, maxAge int) {
	cookie := &http.Cookie{
		Name:     name,
		Domain:   c.domain,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   maxAge,
		Value:    value,
	}
	if maxAge < 0 {
		cookie.Expires = time.Unix(0, 0)
	} else if maxAge > 0 {
		cookie.Expires = globals.Clock.Now().Add(time.Duration(maxAge) * time.Second)
	}
	http.SetCookie(rw, cookie)
}

func (c *cookieStore) Authorize(rw http.ResponseWriter, req *http.Request, user string) error {
	expires := globals.Clock.Now().Add(time.Duration(sessionAge) * time.Second)
	expiresP, _ := ptypes.TimestampProto(expires) // TODO: fix before 9999-12-31
	session := &Session{User: user, Authorized: true, ExpiresAt: expiresP}
	value, err := c.boxer.seal(session, sessionAge)
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
// set in the state cookie's value.
var stateKeyLen = base64.RawURLEncoding.EncodedLen(nonceLen)

func (c *cookieStore) CreateState(rw http.ResponseWriter, redirectURL string) (string, error) {
	if len(redirectURL) > maxRedirectURLLength {
		return "", fmt.Errorf("redirectURL %s... is too long", redirectURL[:maxRedirectURLLength])
	}

	state, err := c.boxer.seal(&OAuthState{RedirectUrl: redirectURL}, stateAge)
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
	if !c.boxer.open(stateKey+cookie.Value, os) {
		return "", fmt.Errorf("invalid state")
	}
	c.setCookie(rw, stateKey, "", -1)
	return os.RedirectUrl, nil
}

type cookieStore struct {
	name   string
	domain string
	boxer  boxer
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
	secretKeyBytes, err := hex.DecodeString(secret)
	if err != nil || len(secretKeyBytes) != 32 {
		var freshKey [32]byte
		if _, err := io.ReadFull(rand.Reader, freshKey[:]); err != nil {
			return nil, err
		}

		return nil, fmt.Errorf(
			"The cookie secret should be a 64-character hex-encoded string.  "+
				"Here's a freshly generated one: %q",
			hex.EncodeToString(freshKey[:]))
	}

	var secretKey [32]byte
	copy(secretKey[:], secretKeyBytes)

	return &cookieStore{
		name:   name,
		domain: domain,
		boxer: boxer{
			secret: secretKey,
			// Default noncer generates a crypto/rand value
			noncer: func() [nonceLen]byte {
				// You must use a different nonce for each message you encrypt with the
				// same key. Since the nonce here is 192 bits long, a random value
				// provides a sufficiently small probability of repeats.
				var nonce [nonceLen]byte
				if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
					panic(err) // don't want to continue encrypting anything
				}
				return nonce
			},
		},
	}, nil
}
