# sohop
[![GoDoc](https://godoc.org/github.com/davars/sohop?status.svg)](https://godoc.org/github.com/davars/sohop)
[![Build Status](https://travis-ci.org/davars/sohop.svg?branch=master)](https://travis-ci.org/davars/sohop)
[![report card](https://goreportcard.com/badge/github.com/davars/sohop)](https://goreportcard.com/report/github.com/davars/sohop)

This program is a reverse proxy that can optionally restrict access to users authenticated with OAuth (currently
supports authorizing members of a specified Github organization, or users whose Google account email matches a
specified regex).  It also provides a health check endpoint that reports the reachability of the upstream services.

I use it to expose erstwhile intranet apps to the public internet while continuing to restrict access, and without
having to configure authentication / authorization in the intranet apps themselves.  They are installed as if they're
still behind a firewall, and sohop handles auth / auth.  This is a configuration that is tilted very much towards the 
usability end of the usability / security spectrum and may not be appropriate for your use case.

## Assumptions

* All outgoing traffic uses HTTPS (HTTP requests are redirected to the HTTPS equivalent URL)
* Each upstream is accessed on a subdomain of the same domain (no path rewriting)
* Upstreams are only accessed via a trusted network.  **WARNING** Since many services in my use case use self-signed
certs, **SSL verification is disabled when communicating with proxied services.**
* Subdomains `health` and `oauth` are reserved
    * `health.<domain>/check` provides a health check endpoint for all proxied services.  
    * `oauth.<domain>/authorize` is used as the oauth callback.
    * `oauth.<domain>/session` shows the user the values in their session.

## Features

* Simple authentication with OAuth
* Automatic TLS certificates via Let's Encrypt
* Proxies WebSocket connections
* HTTP/2 support when compiled with Go >= 1.6
* Replace headers that are forwarded using session cookies and Go templates
* Simple, forkable codebase (maybe not yet but I'd like to get there).  Configure your web server in Go!

## Installation

`go get github.com/davars/sohop/cmd/sohop`

## Usage

```
Usage of sohop:
  -config string
    	Config file (default "config.json")
  -httpAddr string
    	Address to bind HTTP server (default ":80")
  -httpsAddr string
    	Address to bind HTTPS server (default ":443")
```

## Example Configs

```
{
  "Domain": "example.com",
  "Cookie": {
    "Name": "exampleauth",
    "Secret": "3c0767ada2466a92a59c1214061441713aeafe6d115e29aa376c0f9758cdf0f5"
  },  
  "Auth" : {
    "Type": "github-org",
    "Config": {
	  "ClientID": "12345678",
	  "ClientSecret": "12345678",
	  "OrgID": 12345678
	}
  },
  "TLS": {
    "CertFile": "cert.pem",
    "CertKey": "key.pem"
  },
  "Upstreams": {
    "intranet": {
      "URL": "http://10.0.0.16:8888",
      "HealthCheck": "http://10.0.0.16:8888/login",
      "WebSocket": "ws://10.0.0.16:8888",
      "Auth": true,
      "Headers": { "X-WEBAUTH-USER":["{{.Session.Values.user}}"] }
    },
    "public": {
      "URL": "http://10.0.0.16:8111",
      "HealthCheck": "http://10.0.0.16:8111/login.html",
      "WebSocket": "ws://10.0.0.16:8111",
      "Auth": false
    }
  }
}
```


```
{
  "Domain": "example.com",    
  "Auth" : {
    "Type": "gmail-regex",
    "Config": {
	  "Credentials": {"web":{"client_id":"XXXX-yyyyyy.apps.googleusercontent.com","project_id":"example","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://accounts.google.com/o/oauth2/token","auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs","client_secret":"zzzzZZzzZZ","redirect_uris":["https://oauth.example.com/authorized"]}},
	  "EmailRegex":"^davars@gmail.com$"
	}
  },
  "Upstreams": {

...

  }
}
```

The config file id unmarshalled into a sohop.Config struct, described here: https://godoc.org/github.com/davars/sohop#Config

## Testing

```
go test ./...
```

## Roadmap

- [x] Docs
- [x] Tests
- [x] Google Auth (email regex)
- [x] Let's Encrypt provision / renewal
- [ ] Google Auth (Apps domain) (needs advocate)
- [ ] Google Auth (groups) (needs advocate)

## Contributing ##

Contributions welcome! Please fork the repository and open a pull request
with your changes.

## License ##

This is free software, licensed under the ISC license.
