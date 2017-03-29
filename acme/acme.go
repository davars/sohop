// Package acme uses golang.org/x/crypto/acme/autocert to automatically provision TLS certificates.
package acme

import (
	"log"
	"path"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// newTOSCallback returns a function that returns true IFF the agreementURL is contained in the
// agreedTo slice
func newTOSCallback(agreedTo []string) func(string) bool {
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

// Config contains the variables required for autocert
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
	// All files are stored under <DataPath>/autocert.
	DataPath string

	// Domains are the domains for which should be added to provisioned
	// certificates.  The certificate and private key files contain
	// a hash of this collection so that new certificates are provisioned
	// if the list of domains changes.
	Domains []string
}

// Manager returns an *autocert.Manager for this Config
func (c Config) Manager() (*autocert.Manager, error) {
	return &autocert.Manager{
		Prompt:     newTOSCallback(c.TOS),
		HostPolicy: autocert.HostWhitelist(c.Domains...),
		Cache:      autocert.DirCache(path.Join(c.DataPath, "autocert")),
		Email:      c.Email,
		Client:     &acme.Client{DirectoryURL: c.Server},
	}, nil
}
