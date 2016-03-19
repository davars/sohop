// store provices an implementation of sessions.Store that also carries its name around with it.
package store

import (
	"time"

	"encoding/hex"

	"fmt"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

var (
	sessionAge = 24 * time.Hour
)

// Name is a sessions.Store that also has its own name
type Namer interface {
	sessions.Store
	Name() string
}

type namedStore struct {
	*sessions.CookieStore
	domain string
	name   string
	secret string
}

// Name implements the Namer interface, returning this store's name
func (s *namedStore) Name() string {
	return s.name
}

var (
	// ErrMissingName is returned when attempting to create a Namer with an empty name
	ErrMissingName = fmt.Errorf("name cannot be empty")

	// ErrMissingDomain returned when attempting to create a Namer with an empty domain
	ErrMissingDomain = fmt.Errorf("domain cannot be empty")
)

// A KeyError is returned when attmpting to create a Namer with an invalid key.  Keys must be
// hex-encoded strings with length 128.
type KeyError struct {
	sample []byte
}

func (k KeyError) Error() string {
	return fmt.Sprintf(
		"Session store secret should be a 128-character hex-encoded string.  "+
			"Here's a freshly generated one: %q", hex.EncodeToString(k.sample))
}

func (s *namedStore) init() error {
	if s.name == "" {
		return ErrMissingName
	}
	if s.domain == "" {
		return ErrMissingDomain
	}
	key, err := hex.DecodeString(s.secret)
	if err != nil || len(key) != 64 {
		return KeyError{sample: securecookie.GenerateRandomKey(64)}
	}
	cookieStore := sessions.NewCookieStore(key)
	cookieStore.Options.HttpOnly = true
	cookieStore.Options.Secure = true
	cookieStore.Options.Domain = s.domain
	cookieStore.Options.MaxAge = int(sessionAge / time.Second)
	s.CookieStore = cookieStore
	return nil
}

// New returns a new Namer for the given name, secret key and domain.  secret should be a hex-encoded
// string with length 128.
func New(name, secret, domain string) (Namer, error) {
	sc := &namedStore{name: name, domain: domain, secret: secret}
	err := sc.init()
	return sc, err
}
