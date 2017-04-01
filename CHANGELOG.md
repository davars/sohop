# Notable Changes to Sohop

### 2017-04-01

Deprecated flags `certFile` and `certKey` were removed.  These values are now
set in the config file.

Also, the cookie secret is half the length it used to be due to a change in 
the encryption library used.  Running sohop with your old key will cause sohop
to complain and generate a new 64-character hex-encoded secret using 
`crypto/rand`.

### 2017-03-13

Switched ACME support to use `golang.org/x/crypto/acme/autocert`.  There should
be no configuration changes required.  Since there are now individual certs for
each subdomain, and since I trust autocert to reprovision should they expire,
I've disabled the cert check from the health check endpoint when using Acme.

### 2016-04-27

Added support for ACME / Let's Encrypt.  Replace your TLS config with an ACME
config and sohop will figure out which domains to put in your cert from your
Upstreams config.  Watch the logs if you don't know the current Let's Encrypt 
TOS URL.

### 2016-04-23

Moved repository to GitLab

### 2016-04-22

Configuration keys "Github" and "Google" are now deprecated.  They've been 
replaced with a generic "Auth" key.
