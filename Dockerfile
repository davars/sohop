FROM golang:1.17-alpine AS build

WORKDIR /src/
ENV CGO_ENABLED=0

COPY go.* /src/
RUN go mod download
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o /out/sohop ./cmd/sohop

FROM alpine:3.14

RUN apk add --no-cache libcap su-exec

ADD https://raw.githubusercontent.com/cloudflare/cfssl_trust/master/ca-bundle.crt /etc/ssl/certs/ca-certificates.crt
RUN chmod a+r /etc/ssl/certs/ca-certificates.crt

COPY --from=build /out/sohop /usr/local/bin/sohop

# allow sohop to bind to ports 80 & 443 even though it's not root
RUN setcap cap_net_bind_service=+ep /usr/local/bin/sohop

ENTRYPOINT ["/sbin/su-exec", "nobody", "/usr/local/bin/sohop"]

