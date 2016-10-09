package auth

import (
	"errors"
	"net/http"

	"github.com/davars/sohop/store"
	"github.com/gorilla/securecookie"
	"golang.org/x/oauth2"
)

const (
	authorizedKey  = "auth"
	redirectURLKey = "redir"
	stateKey       = "state"
	userKey        = "user"
)

type oauthFLow struct {
	store store.Namer
	auth  Auther
}

func (s *oauthFLow) redirectToLogin(w http.ResponseWriter, r *http.Request) bool {
	session, _ := s.store.Get(r, s.store.Name())
	if auth, ok := session.Values[authorizedKey].(bool); auth && ok {
		return false
	}

	state := string(encodeBase64(securecookie.GenerateRandomKey(30)))

	session.Values[redirectURLKey] = absoluteURL(r)
	session.Values[stateKey] = state
	err := session.Save(r, w)
	if checkServerError(err, w) {
		return true
	}

	url := s.auth.OAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOffline)

	http.Redirect(w, r, url, http.StatusFound)
	return true
}

func (s *oauthFLow) authenticateCode(w http.ResponseWriter, r *http.Request) {
	session, _ := s.store.Get(r, s.store.Name())
	delete(session.Values, authorizedKey)
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, ErrMissingCode, http.StatusBadRequest)
		return
	}

	state, ok := session.Values[stateKey].(string)
	delete(session.Values, stateKey)
	if !ok || state != r.URL.Query().Get("state") {
		checkServerError(errors.New(ErrMissingState), w)
		return
	}

	user, err := s.auth.Auth(code)
	if err != nil {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	session.Values[userKey] = user
	session.Values[authorizedKey] = true
	if checkServerError(err, w) {
		return
	}

	redirectURL, ok := session.Values[redirectURLKey].(string)
	if !ok {
		redirectURL = ""
	}
	delete(session.Values, redirectURLKey)
	if redirectURL == "" {
		checkServerError(errors.New(ErrMissingRedirectURL), w)
		return
	}

	session.Save(r, w)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
