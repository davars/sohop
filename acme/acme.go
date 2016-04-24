package acme

import (
	"log"
	"net/url"
	"path"

	"os"

	"github.com/dkumor/acmewrapper"
)

// newTOSCallback returns an acmewrapper.TOSCallback that returns true IFF the agreementURL is contained in the
// agreedTo slice
func newTOSCallback(agreedTo []string) acmewrapper.TOSCallback {
	return func(agreementURL string) bool {
		log.Printf("Checking if agreement %q is agreed to", agreementURL)
		for _, url := range agreedTo {
			if agreementURL == url {
				return true
			}
		}
		return false
	}
}

type Config struct {
	Server string
	Email  string
	TOS    []string
	RegDir string
}

type Params struct {
	Address string
	Domains []string
}

func (c Config) Wrapper(params *Params) (*acmewrapper.AcmeWrapper, error) {
	u, err := url.Parse(c.Server)
	if err != nil {
		return nil, err
	}
	regPath := path.Join(c.RegDir, u.Host, c.Email)
	err = os.MkdirAll(regPath, 0700)
	if err != nil {
		return nil, err
	}
	return acmewrapper.New(acmewrapper.Config{
		Server:           c.Server,
		Email:            c.Email,
		TOSCallback:      newTOSCallback(c.TOS),
		RegistrationFile: path.Join(regPath, "registration.pem"),
		PrivateKeyFile:   path.Join(regPath, "private_key.pem"),

		Address:     params.Address,
		Domains:     params.Domains,
		TLSCertFile: path.Join(regPath, "cert.pem"),
		TLSKeyFile:  path.Join(regPath, "key.pem"),
	})
}
