package auth

import (
	"fmt"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	githubauth "golang.org/x/oauth2/github"
)

type GithubAuth struct {
	ClientID     string
	ClientSecret string
	OrgID        int
}

func (ga GithubAuth) OAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     ga.ClientID,
		ClientSecret: ga.ClientSecret,
		Scopes:       []string{"user:email", "read:org"},
		Endpoint:     githubauth.Endpoint,
	}
}

func (ga GithubAuth) Authorize(code string) (string, error) {
	oauthConfig := ga.OAuthConfig()

	tok, err := oauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		return "", err
	}

	client := github.NewClient(oauthConfig.Client(oauth2.NoContext, tok))
	user, _, err := client.Users.Get("")
	if err != nil {
		return "", err
	}

	orgs, _, err := client.Organizations.List("", nil)
	if err != nil {
		return "", err
	}

	for _, org := range orgs {
		if *org.ID == ga.OrgID {
			return *user.Login, nil
		}
	}

	return "", fmt.Errorf("unauthorized")
}
