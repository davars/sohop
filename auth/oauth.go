// Package auth implements the OAuth authentication flows for sohop.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/davars/sohop/state"

	"golang.org/x/oauth2"
)

var registeredAuthers = make(map[string]reflect.Type)

// An Auther abstracts an OAuth flow for authenticating and authorizing access
// to handlers
type Auther interface {
	OAuthConfig() *oauth2.Config
	Auth(code string) (string, error)
}

// A Config can be used to create a new Auther
type Config struct {
	// Type is the type of Auther.  Supported types are: github-org
	Type string

	// Config configures the Auther.  The structure of this value varies
	// depending on the auth type.
	Config json.RawMessage
}

// NewAuther returns an Auther for the given Config
func NewAuther(c Config) (Auther, error) {
	configType, ok := registeredAuthers[c.Type]
	if !ok {
		return nil, fmt.Errorf("unknown auther type %q", c.Type)
	}
	config := reflect.New(configType).Interface().(Auther)
	err := json.Unmarshal(c.Config, &config)
	return config, err
}

var (
	// ErrMissingCode is returned if authorization is attempted without an
	// authorization code.
	ErrMissingCode = errors.New("Missing authorization code.")

	// ErrUnauthorized is returned on authorization failure.
	ErrUnauthorized = errors.New("Unauthorized.")
)

// Handler returns an http.Handler that implements whatever authorization steps
// are defined by the Auther (typically exchanging the OAuth2 code for an access
// token and using the token to identify the user).
func Handler(auth Auther, state state.Store) http.Handler {
	flow := &oauthFLow{auth: auth, state: state}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flow.authenticateCode(w, r)
	})
}

// Middleware returns a middleware that checks if the requeset has been
// authorized.  If not, it generates a redirect to the configured Auther login
// URL.
func Middleware(auth Auther, state state.Store) func(http.Handler) http.Handler {
	flow := &oauthFLow{auth: auth, state: state}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if flow.redirectToLogin(w, r) {
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// absoluteURL reconstructs the absolute URL string for the provided request
func absoluteURL(r *http.Request) string {
	proto := "http"
	if r.TLS != nil {
		proto = "https"
	}
	return fmt.Sprintf("%s://%s%s", proto, r.Host, r.RequestURI)
}

// checkServerError renders an http.StatusInternalServerError if the provided
// err is not nil
func checkServerError(err error, w http.ResponseWriter) bool {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return true
	}
	return false
}
