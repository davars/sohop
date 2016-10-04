package auth

import (
	"fmt"
	"reflect"

	"golang.org/x/oauth2"
)

func init() {
	registeredAuthers["mock"] = reflect.TypeOf(MockAuth{})
}

// MockAuth is an Auther that is useful for writing tests
type MockAuth struct {
	ClientID     string
	ClientSecret string
	User         string
	Err          string
}

// OAuthConfig is implemented so MockAuth satisfies the Auther interface.
func (ma MockAuth) OAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     ma.ClientID,
		ClientSecret: ma.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://mock/auth",
			TokenURL: "https://mock/token",
		},
	}
}

// Auth is implemented so MockAuth satisfies the Auther interface.
func (ma MockAuth) Auth(_ string) (string, error) {
	if ma.Err != "" {
		return "", fmt.Errorf(ma.Err)
	}
	return ma.User, nil
}
