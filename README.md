# sohop

This program is a reverse proxy that can optionally restrict access to users belonging to a specific Github organization.  It also provides a health check endpoint that reports the health of the back-end services.

## Assumptions

* All traffic uses HTTPS
* Each back-end has its own subdomain
* Back-ends are only accessed via a trusted network.  **WARNING** Since many services in my use case use self-signed certs, SSL verification is disabled when communicating with proxied services.
* Subdomains `health` and `oauth` are reserved

## Installation

`go get bitbucket.org/davars/sohop/cmd/sohop`

## Usage

```
Usage of sohop:
  -certFile string
    	Server certificate (default "cert.pem")
  -certKey string
    	Server certificate private key (default "key.pem")
  -config string
    	Config file (default "config.json")
  -httpAddr string
    	Address to bind HTTP server (default ":80")
  -httpsAddr string
    	Address to bind HTTPS server (default ":443")
```

## Example Config

```
{
  "Domain": "example.com",
  "AuthorizedOrgId": 12345678,
  "GithubApi":{
    "ClientID": "12345678",
    "ClientSecret": "12345678"
  },
  "Backends": {
    "intranet": {
      "URL": "http://10.0.0.16:8888",
      "HealthCheck": "http://10.0.0.16:8888/health",
      "WebSocket": "ws://10.0.0.16:8888",
      "Auth": true
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

## Testing

`TODO`

## Roadmap

- [ ] Docs
- [ ] Tests
- [ ] Google Auth (email regex)
- [ ] Let's Encrypt provision / renewal
- [ ] Google Auth (groups)

## Contributing ##

Contributions welcome! Please fork the repository and open a pull request
with your changes.

## License ##

This is free software, licensed under the ISC license.
