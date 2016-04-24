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

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"gitlab.com/davars/sohop/acme"
	"gitlab.com/davars/sohop/auth"
	"gitlab.com/davars/sohop/store"
)

type Config struct {
	Domain          string
	Upstreams       map[string]upstreamSpec
	Auth            auth.Config
	AuthorizedOrgID int
	Cookie          CookieConfig
	TLS             TLSConfig
	Acme            *acme.Config

	Github *auth.GithubAuth
	Google *auth.GoogleAuth
}

type CookieConfig struct {
	Name   string
	Secret string
}

type TLSConfig struct {
	CertFile string
	CertKey  string
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
	s.Config.checkTLS()

	go func() {
		if s.Config.Acme == nil {
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
			w, err := s.Config.Acme.Wrapper(&acme.Params{
				Address: s.HTTPSAddr,
				Domains: domains,
			})
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

type upstreamSpec struct {
	URL         string
	Auth        bool
	HealthCheck string
	WebSocket   string
	Headers     http.Header
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
