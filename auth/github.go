package auth

import (
	"fmt"
	"reflect"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	githubauth "golang.org/x/oauth2/github"
)

func init() {
	registeredAuthers["github-org"] = reflect.TypeOf(GithubAuth{})
}

// GithubAuth implements the Github OrgID middleware.  Users must be logged into
// Github and be a member of the specified Org to be authorized.
//
// To use, you'll need to create an application to use the Github API for
// authentication.  Read https://developer.github.com/guides/basics-of-authentication/
// to get an overview for how this works.
type GithubAuth struct {
	ClientID     string
	ClientSecret string

	// OrgID is the ID of the org whose members are authorized. Run
	// `curl https://api.github.com/orgs/:org` to get the id.
	OrgID int
}

func (ga GithubAuth) OAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     ga.ClientID,
		ClientSecret: ga.ClientSecret,
		Scopes:       []string{"user:email", "read:org"},
		Endpoint:     githubauth.Endpoint,
	}
}

func (ga GithubAuth) Auth(code string) (string, error) {
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
