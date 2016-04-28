package acme

import (
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"

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

// Config contains the variables required for AcmeWrapper
type Config struct {
	// Server is the ACME server to use
	Server string

	// Email is the account owner's email
	Email string

	// TOS is an array containing the URLs for accepted Terms of Service
	TOS []string

	// DataPath is the path where files (registration, registration private key,
	// cert, and cert key) should be stored.
	//
	// All files are stored under <DataPath>/<ACME server host>/<account email>/.
	DataPath string

	// Domains are the domains for which should be added to provisioned
	// certificates.  The certificate and private key files contain
	// a hash of this collection so that new certificates are provisioned
	// if the list of domains changes.
	Domains []string

	// Address is the address the server is listening on.  Set automatically.
	Address string `json:"-"`
}

// Wrapper returns an AcmeWrapper for this Config
func (c Config) Wrapper() (*acmewrapper.AcmeWrapper, error) {
	u, err := url.Parse(c.Server)
	if err != nil {
		return nil, err
	}

	regPath := path.Join(c.DataPath, u.Host, c.Email)
	err = os.MkdirAll(regPath, 0700)
	if err != nil {
		return nil, err
	}

	sort.Strings(c.Domains)
	h := sha1.New()
	io.WriteString(h, strings.Join(c.Domains, "-"))
	domainHash := fmt.Sprintf("%x", h.Sum(nil))[0:10]

	return acmewrapper.New(acmewrapper.Config{
		Server:           c.Server,
		Email:            c.Email,
		TOSCallback:      newTOSCallback(c.TOS),
		RegistrationFile: path.Join(regPath, "registration.pem"),
		PrivateKeyFile:   path.Join(regPath, "private_key.pem"),

		Address:     c.Address,
		Domains:     c.Domains,
		TLSCertFile: path.Join(regPath, fmt.Sprintf("cert-%s.pem", domainHash)),
		TLSKeyFile:  path.Join(regPath, fmt.Sprintf("key-%s.pem", domainHash)),
	})
}
