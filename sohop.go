package sohop

import (
	"crypto/rand"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"time"

	"bitbucket.org/davars/sohop/auth"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

type Config struct {
	Domain          string
	Upstreams       map[string]upstreamSpec
	Github          *auth.GithubAuth
	Google          *auth.GoogleAuth
	AuthorizedOrgID int
}

type Server struct {
	Config    *Config
	CertFile  string
	CertKey   string
	HTTPAddr  string
	HTTPSAddr string

	proxy http.Handler
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func (s Server) Run() {
	var err error

	store.Options.HttpOnly = true
	store.Options.Secure = true
	store.Options.Domain = s.Config.Domain
	store.Options.MaxAge = int(sessionAge / time.Second)

	go func() {
		err = http.ListenAndServeTLS(s.HTTPSAddr, s.CertFile, s.CertKey, s.handler())
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

var (
	store       = sessions.NewCookieStore(securecookie.GenerateRandomKey(64))
	sessionName = sessionID()
	sessionAge  = 24 * time.Hour
)

func sessionID() string {
	n, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		log.Fatal(err)
	}
	id := fmt.Sprintf("_s%d", n)
	return id
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
	oauthRouter.Path("/authorized").Handler(auth.Handler(store, sessionName, conf.authorizer()))
	authenticating := auth.Middleware(store, sessionName, conf.authorizer())

	// TODO: switch to JWT so that this isn't necessary
	oauthRouter.Path("/session").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, sessionName)
		fmt.Fprintf(w, "%v", session.Values)
	})

	healthRouter := router.Host(fmt.Sprintf("health.%s", conf.Domain)).Subrouter()
	healthRouter.Path("/check").Handler(s.HealthHandler())

	proxyRouter := router.Host(fmt.Sprintf("{subdomain:[a-z]+}.%s", conf.Domain)).Subrouter()
	proxy := s.ProxyHandler()
	proxyRouter.MatcherFunc(requiresAuth(conf)).Handler(authenticating(proxy))
	proxyRouter.PathPrefix("/").Handler(proxy)

	return logging(router)
}
