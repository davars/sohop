package auth

import (
	"fmt"
	"reflect"

	"golang.org/x/oauth2"
)

func init() {
	registeredAuthorizers["mock"] = reflect.TypeOf(MockAuth{})
}

type MockAuth struct {
	ClientID     string
	ClientSecret string
	User         string
	Err          string
}

func (ma MockAuth) OAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     ma.ClientID,
		ClientSecret: ma.ClientSecret,
	}
}

func (ma MockAuth) Authorize(code string) (string, error) {
	if ma.Err != "" {
		return "", fmt.Errorf(ma.Err)
	}
	return ma.User, nil
}
