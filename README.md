# sohop

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
* Proxies WebSocket connections
* Replace headers that are forwarded using session cookies and Go templates
* Simple, forkable codebase (maybe not yet but I'd like to get there).  Configure your web server in Go!

## Installation

`go get bitbucket.org/davars/sohop/cmd/sohop`

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
    "Secret": "27e21c8d866594bd446c4a509d890ce2f59dcb26d89751b77ca236e5be3e0d7c26532a60e1ed9fd4f7b924e363d64e7a44a56dd57d84cf34eb7f0db0e19889f5"
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


### Config Definitions

**Domain**
: The domain to which the subdomains belong.  Also used as the domain for the session cookie.

**Cookie**
: (optional) An object, configures the session cookie store.  Members are defined below.

**Cookie.Name**
: (optional) Name of the session cookie.  If not set, a random name will be generated on start-up.

**Cookie.Secret**
: (optional) Secret key used to authenticate session cookies. Should be a hex-encoded string 128 characters in length (64 byte key).  If not set, a random key will be generated on start-up.  Run `openssl rand -hex 64` to generate a key.

**Auth**
: An object, configures authentication.

**Auth.Type**
: Supported types are: github-org, google-regex

**Auth.Config**
: The structure of this value varies depending on the auth type.

**[github-org] Auth.Config.ClientID** / **Auth.Config.ClientSecret**
: You'll need to create an application to use the Github API for authentication.  Read Github's [Basics of Authentication](https://developer.github.com/guides/basics-of-authentication/) to get an overview for how this works.

**[github-org] Auth.Config.OrgID**
: ID of the org to allow access. Run `curl https://api.github.com/orgs/:org` to get the id.

**[google-regex] Auth.Config.Credentials**
: An object in the same format as can be downloaded from the Google Developers Console.  See [google.ConfigFromJSON](https://godoc.org/golang.org/x/oauth2/google#ConfigFromJSON) for more info.

**[google-regex] Auth.Config.EmailRegex**
: Allow users whose email matches the regex access to authenticated upstream servers.  Be careful, and keep it simple.

**TLS**
: An object, specifies the TLS configuration for the server.  Memebrs are defined below.

**TLS.CertFile**
: Path to the PEM-encoded server certificate.

**TLS.CertKey**
: Path to the unencrypted PEM-encoded private key for the server certificate.

**Upstreams**
: An array of configurations for upstream servers.  Keys are the subdomain to proxy to the configured server.  Values are objects whose members are defined below.

**Upstreams.URL**
: The URL of the upstream server.

**Upstreams.HealthCheck**
: (optional) URL to use as a health check, if different from Upstreams.URL (for example if Upstreams.URL returns a 302 response).  Should return a 200 response if the upstream is healthy.

**Upstreams.WebSocket**
: (optional) If provided, sohop will also proxy WebSocket connections to this URL.

**Upstreams.Auth**
: (default: false) Require authentication for this upstream.

**Upstreams.Headers**
: (optional) A map of headers to explicitly set on the upstream request.  Can be a template, evaluated with the current session available as `.Session`


## Testing

```
go test ./...
```

## Roadmap

- [x] Docs
- [x] Tests
- [x] Google Auth (email regex)
- [ ] Let's Encrypt provision / renewal
- [ ] Google Auth (Apps domain) (needs advocate)
- [ ] Google Auth (groups) (needs advocate)

## Rant

There seems to be a trend where all config file formats trend towards Turing-completeness over time.  Life is too short for
understanding the directives of yet another arbitrary config file format, and still not being free from having to
patch the software anyway when it falls just short of your needs.  I'd rather have all of Go available to me when
'configuring' my web server so that I can perform truly arbitrary processing on requests.

## Contributing ##

Contributions welcome! Please fork the repository and open a pull request
with your changes.

## License ##

This is free software, licensed under the ISC license.