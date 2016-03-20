package sohop

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"

	"bitbucket.org/davars/sohop/auth"
	"bitbucket.org/davars/sohop/store"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
)

type Config struct {
	Domain          string
	Upstreams       map[string]upstreamSpec
	Github          *auth.GithubAuth
	Google          *auth.GoogleAuth
	AuthorizedOrgID int
	Cookie          CookieConfig
	TLS             TLSConfig
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
		err = http.ListenAndServeTLS(s.HTTPSAddr, s.Config.TLS.CertFile, s.Config.TLS.CertKey, s.handler())
		check(err)
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

func (c *Config) authorizer() auth.Authorizer {
	if c.Github != nil && c.Google != nil {
		log.Fatal("can only use one authorizer; please configure either Google or Github authorization")
	}
	if c.Github == nil && c.Google == nil {
		log.Fatal("must define an authorizer; please configure either Google or Github authorization")
	}
	if c.Github != nil {
		return c.Github
	}
	return c.Google
}

func (s Server) handler() http.Handler {
	router := mux.NewRouter()
	router.NotFoundHandler = http.HandlerFunc(notFound)

	conf := s.Config
	oauthRouter := router.Host(fmt.Sprintf("oauth.%s", conf.Domain)).Subrouter()
	oauthRouter.Path("/authorized").Handler(auth.Handler(s.store, conf.authorizer()))
	authenticating := auth.Middleware(s.store, conf.authorizer())

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
