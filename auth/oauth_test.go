package auth

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func TestNewAuther(t *testing.T) {
	tests := []struct {
		in  Config
		out Auther
		err string
	}{
		{
			in: Config{
				Type:   "none",
				Config: []byte{},
			},
			err: `unknown auther type "none"`,
		},
		{
			in: Config{
				Type:   "github-org",
				Config: []byte(`{"ClientID": "id", "ClientSecret": "secret", "OrgID": 42}`),
			},
			out: &GithubAuth{ClientID: "id", ClientSecret: "secret", OrgID: 42},
		},
		{
			in: Config{
				Type: "gmail-regex",
				Config: []byte(`{
				"Credentials": {"web":{"client_id":"client-id","project_id":"example","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://accounts.google.com/o/oauth2/token","auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs","client_secret":"client-secret","redirect_uris":["https://oauth.example.com/authorized"]}},
				"EmailRegex":"^test@example.com$"
				}`),
			},
			out: &GoogleAuth{
				config: &oauth2.Config{
					ClientID:     "client-id",
					ClientSecret: "client-secret",
					Endpoint:     google.Endpoint,
					RedirectURL:  "https://oauth.example.com/authorized",
					Scopes:       []string{"openid", "email"},
				},
				emailRegex: regexp.MustCompile("^test@example.com$"),
			},
		},
		{
			in: Config{
				Type:   "mock",
				Config: []byte(`{"ClientID": "id", "ClientSecret": "secret", "User": "user", "Err": "error"}`),
			},
			out: &MockAuth{ClientID: "id", ClientSecret: "secret", User: "user", Err: "error"},
		},
		{
			in: Config{
				Type:   "gmail-regex",
				Config: []byte(`{}`),
			},
			out: &GoogleAuth{},
			err: "unexpected end of JSON input",
		},
	}

	for _, test := range tests {
		auth, err := NewAuther(test.in)
		require.Equal(t, test.out, auth)
		if test.err == "" {
			require.NoError(t, err)
		} else {
			require.Equal(t, test.err, err.Error())
		}
	}
}
