package sohop

import (
	"bytes"
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/yhat/wsutil"
)

type headerTemplate map[string][]*template.Template

type upstream struct {
	HTTPProxy       *httputil.ReverseProxy
	WSProxy         *wsutil.ReverseProxy
	headerTemplates headerTemplate
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
		templates := make(headerTemplate, len(spec.Headers))
		for k, v := range spec.Headers {
			for _, t := range v {
				template := template.Must(template.New("").Parse(t))
				templates[k] = append(templates[k], template)
			}
		}
		upstream.headerTemplates = templates

		m[name] = upstream
	}

	return m, nil
}

// ProxyHandler selects the appropriate upstream based on subdomain of the
// incoming request and does the proxying.
func (s Server) ProxyHandler() http.Handler {
	upstreams, err := s.Config.createUpstreams()
	check(err)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subdomain := mux.Vars(r)["subdomain"]
		upstream, ok := upstreams[subdomain]
		if !ok {
			notFound(w, r)
			return
		}

		if len(upstream.headerTemplates) > 0 {
			session, _ := s.store.Get(r, s.store.Name())
			for k, vs := range upstream.headerTemplates {
				r.Header.Del(k)
				for _, v := range vs {
					buf := &bytes.Buffer{}
					err := v.Execute(buf, struct{ Session *sessions.Session }{Session: session})
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					r.Header.Add(k, buf.String())
				}
			}
		}

		if upstream.WSProxy != nil && wsutil.IsWebSocketRequest(r) {
			// HACK: EdgeOS treats headers as case-sensitive.  Bypass canonicalization.
			for k, v := range r.Header {
				if strings.Contains(k, "Websocket") {
					fixed := strings.Replace(k, "Websocket", "WebSocket", -1)
					r.Header[fixed] = v
					delete(r.Header, k)
				}
			}

			upstream.WSProxy.ServeHTTP(w, r)
			return
		}

		if upstream.HTTPProxy != nil {
			upstream.HTTPProxy.ServeHTTP(w, r)
			return
		}

		notFound(w, r)
	})
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
