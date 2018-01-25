// Package sohop implements an OAuth-authenticating reverse proxy.
package sohop

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/davars/sohop/acme"
	"github.com/davars/sohop/auth"
	"github.com/davars/sohop/state"
	"github.com/golang/protobuf/jsonpb"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/acme/autocert"
)

// A cookieStore can be used to set up a sohop proxy
type Config struct {
	// Domain is the domain to which the subdomains belong. Also used as the
	// domain for the session cookie.
	Domain string

	// Upstreams is an array of configurations for upstream servers.  Keys are
	// the subdomain to proxy to the configured server.  GetSession describe
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

// A Server is an OAuth-authenticating reverse proxy.
type Server struct {
	Config    *Config
	HTTPAddr  string
	HTTPSAddr string

	proxy       http.Handler
	health      *healthReport
	storeConfig state.Store
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Run bootstraps the listeners then waits forever.
func (s Server) Run() {
	var err error

	go func() {
		s.health = &healthReport{}
		for {
			s.performCheck()
			time.Sleep(5 * time.Second)
		}
	}()

	var m *autocert.Manager
	if s.Config.Acme != nil {
		domains := make([]string, 0, len(s.Config.Upstreams)+2)
		for _, subdomain := range []string{"oauth", "health"} {
			domains = append(domains, fmt.Sprintf("%s.%s", subdomain, s.Config.Domain))
		}
		for subdomain := range s.Config.Upstreams {
			domains = append(domains, fmt.Sprintf("%s.%s", subdomain, s.Config.Domain))
		}

		s.Config.Acme.Domains = domains

		m, err = s.Config.Acme.Manager()
		check(err)
	}
	go func() {
		if m == nil {
			s.Config.checkTLS()
			err = http.ListenAndServeTLS(s.HTTPSAddr, s.Config.TLS.CertFile, s.Config.TLS.CertKey, s.handler())
			check(err)
		} else {
			tlsConfig := &tls.Config{
				GetCertificate: m.GetCertificate,
				NextProtos:     []string{"h2"},
			}

			server := &http.Server{
				Addr:      s.HTTPSAddr,
				Handler:   s.handler(),
				TLSConfig: tlsConfig,
			}

			err = server.ListenAndServeTLS("", "")
			check(err)
		}

	}()
	go func() {
		var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Scheme = "https"
			r.URL.Host = r.Host + s.HTTPSAddr
			http.Redirect(w, r, r.URL.String(), http.StatusMovedPermanently)
			return
		})

		if m != nil {
			handler = m.HTTPHandler(handler)
		}

		err := http.ListenAndServe(s.HTTPAddr, handler)
		check(err)
	}()
	select {}
}

// UpstreamConfig configures a single upstream endpoint.
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

	// Headers can be used to replace the headers of an incoming request
	// before it is sent upstream.  The values are templates, evaluated with the
	// current session available as `.Session`.
	Headers http.Header
}

func (c *Config) storeConfig() state.Store {
	if c.Cookie.Name == "" {
		n, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
		if err != nil {
			log.Fatal(err)
		}
		c.Cookie.Name = fmt.Sprintf("_s%d", n)
	}
	if c.Cookie.Secret == "" {
		var key [32]byte
		if _, err := io.ReadFull(rand.Reader, key[:]); err != nil {
			panic(err) // don't want to continue encrypting anything
		}
		c.Cookie.Secret = hex.EncodeToString(key[:])
	}
	conf, err := state.New(c.Cookie.Name, c.Cookie.Secret, c.Domain)
	if err != nil {
		log.Fatal(err)
	}
	return conf
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

var authorizedTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
	<head>
	<meta charset="UTF-8">
	<title>Authorized</title>
	</head>
	<body>
		<p>Authorized.  Continue to <a href="{{.}}">{{.}}</a>?</p>
	</body>
</html>
`))

func (s Server) handler() http.Handler {
	router := mux.NewRouter()
	router.NotFoundHandler = http.HandlerFunc(notFound)

	conf := s.Config
	oauthRouter := router.Host(fmt.Sprintf("oauth.%s", conf.Domain)).Subrouter()

	auther := conf.auther()
	s.storeConfig = conf.storeConfig()
	oauthRouter.Path("/authorized").Handler(auth.Handler(auther, s.storeConfig))
	authenticating := auth.Middleware(auther, s.storeConfig)

	oauthRouter.Path("/session").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		(&jsonpb.Marshaler{Indent: "  "}).Marshal(w, s.storeConfig.GetSession(r))
	})

	oauthRouter.Path("/auth").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.storeConfig.IsAuthorized(r) {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
	})

	oauthRouter.Path("/signin").Queries("rd", "{rd}").Handler(authenticating(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rd := mux.Vars(r)["rd"]
		authorizedTemplate.Execute(w, rd)
	})))

	healthRouter := router.Host(fmt.Sprintf("health.%s", conf.Domain)).Subrouter()
	healthRouter.Path("/check").Handler(s.HealthHandler())

	proxyRouter := router.Host(fmt.Sprintf("{subdomain:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?}.%s", conf.Domain)).Subrouter()
	proxy := s.ProxyHandler()
	proxyRouter.MatcherFunc(requiresAuth(conf)).Handler(authenticating(proxy))
	proxyRouter.PathPrefix("/").Handler(proxy)

	return logging(router)
}
