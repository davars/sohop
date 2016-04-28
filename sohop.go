// Package sohop implements an OAuth-authenticating reverse proxy.
package sohop

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"

	"github.com/davars/sohop/acme"
	"github.com/davars/sohop/auth"
	"github.com/davars/sohop/store"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
)

// A Config can be used to set up a sohop proxy
type Config struct {
	// Domain is the domain to which the subdomains belong. Also used as the
	// domain for the session cookie.
	Domain string

	// Upstreams is an array of configurations for upstream servers.  Keys are
	// the subdomain to proxy to the configured server.  Values describe
	// various aspects of the upstream server.
	Upstreams map[string]UpstreamConfig

	// Auth configures the auth middleware.
	Auth auth.Config

	// Cookie configures the session cookie store.
	Cookie CookieConfig

	// TLS can be used to specify a static TLS configuration for the server.
	// It is overridden by the values from the AcmeWrapper if Acme is used.
	TLS TLSConfig

	// Acme configures automatic provisioning and renewal of TLS certificates
	// using the ACME protocol.
	Acme *acme.Config

	// Deprecated.  See https://godoc.org/github.com/davars/sohop/auth#Config.
	Github *auth.GithubAuth

	// Deprecated.  See https://godoc.org/github.com/davars/sohop/auth#Config.
	Google *auth.GoogleAuth
}

// CookieConfig configures the session cookie store.
type CookieConfig struct {
	// Name is the name of the session cookie.  If not set, a random name will
	// be generated on start-up.
	Name string

	// Secret is the private key used to authenticate session cookies. Should be
	// a hex-encoded string 128 characters in length (64 byte key).  If not set,
	// a random key will be generated on start-up.  Run `openssl rand -hex 64`
	// to generate a key.
	Secret string
}

// TLSConfig configures the server certificate.
type TLSConfig struct {
	// CertFile is a path to the PEM-encoded server certificate.
	CertFile string

	// CertKey is a path to the unencrypted PEM-encoded private key for the
	// server certificate.
	CertKey string
}

type Server struct {
	Config    *Config
	HTTPAddr  string
	HTTPSAddr string

	proxy http.Handler
	store store.Namer
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func (s Server) Run() {
	var err error

	s.store, err = s.Config.namer()
	check(err)

	go func() {
		if s.Config.Acme == nil {
			s.Config.checkTLS()
			err = http.ListenAndServeTLS(s.HTTPSAddr, s.Config.TLS.CertFile, s.Config.TLS.CertKey, s.handler())
			check(err)
		} else {
			domains := make([]string, 0, len(s.Config.Upstreams)+2)
			for _, subdomain := range []string{"oauth", "health"} {
				domains = append(domains, fmt.Sprintf("%s.%s", subdomain, s.Config.Domain))
			}
			for subdomain := range s.Config.Upstreams {
				domains = append(domains, fmt.Sprintf("%s.%s", subdomain, s.Config.Domain))
			}

			s.Config.Acme.Address = s.HTTPSAddr
			s.Config.Acme.Domains = domains

			w, err := s.Config.Acme.Wrapper()
			check(err)

			s.Config.TLS.CertFile = w.Config.TLSCertFile
			s.Config.TLS.CertKey = w.Config.TLSKeyFile

			tlsConfig := w.TLSConfig()

			listener, err := tls.Listen("tcp", s.HTTPSAddr, tlsConfig)
			check(err)

			// To enable http2, we need http.Server to have reference to tlsconfig
			// https://github.com/golang/go/issues/14374
			server := &http.Server{
				Addr:      s.HTTPSAddr,
				Handler:   s.handler(),
				TLSConfig: tlsConfig,
			}
			err = server.Serve(listener)
			check(err)
		}
	}()
	go func() {
		err := http.ListenAndServe(s.HTTPAddr,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.URL.Scheme = "https"
				r.URL.Host = r.Host + s.HTTPSAddr
				http.Redirect(w, r, r.URL.String(), http.StatusMovedPermanently)
				return
			}))
		check(err)
	}()
	select {}
}

type UpstreamConfig struct {
	// The URL of the upstream server.
	URL string

	// Auth is whether requests to this upstream require authentication.
	Auth bool

	// HealthCheck is a URL to use as a health check, if different from
	// Upstreams.URL (for example if UpstreamConfig.URL returns a 302 response).
	// It should return a 200 response if the upstream is healthy.
	HealthCheck string

	// WebSocket is a ws:// or wss:// URL receive proxied WebSocket connections.
	WebSocket string

	// Headers can be used to replace the headers of an incomping request
	// before it is sent upstream.  The values are templates, evaluated with the
	// current session available as `.Session`.
	Headers http.Header
}

func (c *Config) namer() (store.Namer, error) {
	if c.Cookie.Name == "" {
		n, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
		if err != nil {
			log.Fatal(err)
		}
		c.Cookie.Name = fmt.Sprintf("_s%d", n)
	}
	if c.Cookie.Secret == "" {
		c.Cookie.Secret = hex.EncodeToString(securecookie.GenerateRandomKey(64))
	}
	return store.New(c.Cookie.Name, c.Cookie.Secret, c.Domain)
}

func (c *Config) checkTLS() {
	if _, err := os.Stat(c.TLS.CertFile); err != nil {
		log.Fatalf("cannot find TLS.CertFile: %v", err)
	}
	if _, err := os.Stat(c.TLS.CertKey); err != nil {
		log.Fatalf("cannot find TLS.CertKey: %v", err)
	}
}

func (c *Config) auther() auth.Auther {
	if c.Github != nil || c.Google != nil {
		log.Fatal("Authorization configuration has changed.  Refer to the README regarding the \"Auth\" key.")
	}
	a, err := auth.NewAuther(c.Auth)
	if err != nil {
		log.Fatalf("NewAuther: %v", err)
	}
	return a
}

func (s Server) handler() http.Handler {
	router := mux.NewRouter()
	router.NotFoundHandler = http.HandlerFunc(notFound)

	conf := s.Config
	oauthRouter := router.Host(fmt.Sprintf("oauth.%s", conf.Domain)).Subrouter()
	auther := conf.auther()
	oauthRouter.Path("/authorized").Handler(auth.Handler(s.store, auther))
	authenticating := auth.Middleware(s.store, auther)

	// TODO: switch to JWT so that this isn't necessary
	oauthRouter.Path("/session").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := s.store.Get(r, s.store.Name())
		fmt.Fprintf(w, "%v", session.Values)
	})

	healthRouter := router.Host(fmt.Sprintf("health.%s", conf.Domain)).Subrouter()
	healthRouter.Path("/check").Handler(s.HealthHandler())

	proxyRouter := router.Host(fmt.Sprintf("{subdomain:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?}.%s", conf.Domain)).Subrouter()
	proxy := s.ProxyHandler()
	proxyRouter.MatcherFunc(requiresAuth(conf)).Handler(authenticating(proxy))
	proxyRouter.PathPrefix("/").Handler(proxy)

	return logging(router)
}
