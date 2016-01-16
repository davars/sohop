# sohop

This program is a reverse proxy that can optionally restrict access to users belonging to a specific Github organization.  It also provides a health check endpoint that reports the health of the back-end services.

## Assumptions

* All traffic uses HTTPS
* Each back-end has its own subdomain
* Back-ends are only accessed via a trusted network.  **WARNING** Since many services in my use case use self-signed certs, **SSL verification is disabled when communicating with proxied services.**
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

## Example Configs

```
{
  "Domain": "example.com",
  "Github":{
    "ClientID": "12345678",
    "ClientSecret": "12345678",
    "OrgID": 12345678
  },
  "Backends": {
    "intranet": {
      "URL": "http://10.0.0.16:8888",
      "HealthCheck": "http://10.0.0.16:8888/tree",
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

Credentials are the same format as can be downloaded from the Google Developers Console.
See [google.ConfigFromJSON](https://godoc.org/golang.org/x/oauth2/google#ConfigFromJSON) for more info.
```
{
  "Domain": "example.com",
  "Google":{
    "Credentials": {"web":{"client_id":"XXXX-yyyyyy.apps.googleusercontent.com","project_id":"example","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://accounts.google.com/o/oauth2/token","auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs","client_secret":"zzzzZZzzZZ","redirect_uris":["https://oauth.example.com/authorized"]}},
    "EmailRegex":"^davars@gmail.com$"
  },
  "Backends": {

...

  }
}
```

## Testing

`TODO`

## Roadmap

- [ ] Docs
- [ ] Tests
- [x] Google Auth (email regex)
- [ ] Let's Encrypt provision / renewal
- [ ] Google Auth (Apps domain)
- [ ] Google Auth (groups)

## Contributing ##

Contributions welcome! Please fork the repository and open a pull request
with your changes.

## License ##

This is free software, licensed under the ISC license.
