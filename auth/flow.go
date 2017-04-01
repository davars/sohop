package auth

import (
	"net/http"

	"github.com/davars/sohop/state"
	"golang.org/x/oauth2"
)

type oauthFLow struct {
	auth  Auther
	state state.Store
}

func (s *oauthFLow) redirectToLogin(w http.ResponseWriter, r *http.Request) bool {
	if s.state.IsAuthorized(r) {
		return false
	}

	state, err := s.state.CreateState(w, absoluteURL(r))
	if checkServerError(err, w) {
		return true
	}

	url := s.auth.OAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusFound)
	return true
}

func (s *oauthFLow) authenticateCode(w http.ResponseWriter, r *http.Request) {
	redirectURL, err := s.state.RedeemState(w, r, r.URL.Query().Get("state"))
	if checkServerError(err, w) {
		return
	}

	if s.state.IsAuthorized(r) {
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, ErrMissingCode.Error(), http.StatusBadRequest)
		return
	}

	user, err := s.auth.Auth(code)
	if err != nil {
		http.Error(w, ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}
	if err := s.state.Authorize(w, r, user); err != nil {
		http.Error(w, ErrUnauthorized.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
