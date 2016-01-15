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

func (c *Config) proxies() (map[string]*httputil.ReverseProxy,
	map[string]*wsutil.ReverseProxy,
	error) {
	// Assume backends are accessible via trusted network
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	proxyMap := make(map[string]*httputil.ReverseProxy)
	wsProxyMap := make(map[string]*wsutil.ReverseProxy)

	for name, backend := range c.Backends {
		if backend.URL != "" {
			target, err := url.Parse(backend.URL)
			if err != nil {
				return nil, nil, err
			}
			proxyMap[name] = httputil.NewSingleHostReverseProxy(target)
			proxyMap[name].Transport = transport
		}

		if backend.WebSocket != "" {
			target, err := url.Parse(backend.WebSocket)
			if err != nil {
				return nil, nil, err
			}
			wsProxyMap[name] = wsutil.NewSingleHostReverseProxy(target)
		}
	}

	return proxyMap, wsProxyMap, nil
}

func (c *Config) ProxyHandler() (http.Handler, error) {
	backendProxies, webSocketProxies, err := c.proxies()
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subdomain := mux.Vars(r)["subdomain"]
		if wsutil.IsWebSocketRequest(r) {
			if proxy, ok := webSocketProxies[subdomain]; ok {
				proxy.ServeHTTP(w, r)
			} else {
				notFound(w, r)
			}
			return
		}

		if proxy, ok := backendProxies[subdomain]; ok {
			proxy.ServeHTTP(w, r)
		} else {
			notFound(w, r)
		}
	}), nil
}

func requiresAuth(c *Config) mux.MatcherFunc {
	return func(r *http.Request, rm *mux.RouteMatch) bool {
		subdomain := strings.Split(r.Host, ".")[0]
		if backend, ok := c.Backends[subdomain]; ok {
			return backend.Auth
		}

		return true
	}
}
