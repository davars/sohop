FROM alpine:3.4
MAINTAINER Dave Jack <davars@gmail.com>

RUN apk add --no-cache libcap su-exec

ADD https://raw.githubusercontent.com/cloudflare/cfssl_trust/master/ca-bundle.crt /etc/ssl/certs/ca-certificates.crt
RUN chmod a+r /etc/ssl/certs/ca-certificates.crt

COPY sohop /usr/local/bin/sohop

# allow sohop to bind to ports 80 & 443 even though it's not root
RUN setcap cap_net_bind_service=+ep /usr/local/bin/sohop

ENTRYPOINT ["/sbin/su-exec", "nobody", "/usr/local/bin/sohop"]