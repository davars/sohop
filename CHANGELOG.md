# Notable Changes to Sohop

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
