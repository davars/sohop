package sohop

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	"github.com/yhat/wsutil"
)

type upstream struct {
	HTTPProxy *httputil.ReverseProxy
	WSProxy   *wsutil.ReverseProxy
}

func (c *Config) createUpstreams() (map[string]upstream, error) {
	// Assume upstreams are accessible via trusted network
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	m := map[string]upstream{}

	for name, spec := range c.Upstreams {
		upstream := upstream{}

		if spec.URL != "" {
			target, err := url.Parse(spec.URL)
			if err != nil {
				return nil, err
			}
			upstream.HTTPProxy = httputil.NewSingleHostReverseProxy(target)
			upstream.HTTPProxy.Transport = transport
		}

		if spec.WebSocket != "" {
			target, err := url.Parse(spec.WebSocket)
			if err != nil {
				return nil, err
			}
			upstream.WSProxy = wsutil.NewSingleHostReverseProxy(target)
			upstream.WSProxy.TLSClientConfig = tlsConfig
		}

		m[name] = upstream
	}

	return m, nil
}

func (c *Config) ProxyHandler() (http.Handler, error) {
	upstreams, err := c.createUpstreams()
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subdomain := mux.Vars(r)["subdomain"]
		upstream, ok := upstreams[subdomain]
		if !ok {
			notFound(w, r)
			return
		}
		if upstream.WSProxy != nil && wsutil.IsWebSocketRequest(r) {
			upstream.WSProxy.ServeHTTP(w, r)
			return
		}

		if upstream.HTTPProxy != nil {
			upstream.HTTPProxy.ServeHTTP(w, r)
			return
		}

		notFound(w, r)
	}), nil
}

func requiresAuth(c *Config) mux.MatcherFunc {
	return func(r *http.Request, rm *mux.RouteMatch) bool {
		subdomain := strings.Split(r.Host, ".")[0]
		if upstream, ok := c.Upstreams[subdomain]; ok {
			return upstream.Auth
		}

		return true
	}
}
