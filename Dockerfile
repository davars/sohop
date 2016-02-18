FROM golang:1.6
MAINTAINER Dave Jack <davars@gmail.com>

# explicitly set user/group IDs
RUN groupadd -r swuser -g 1000 && \
    useradd -u 1000 -r -g swuser -d /home/swuser -s /sbin/nologin -c "Docker image user" swuser && \
    mkdir -p /home/swuser && \
    chown -R swuser:swuser /home/swuser

# grab gosu for easy step-down from root
RUN gpg --keyserver pool.sks-keyservers.net --recv-keys B42F6819007F00F88E364FD4036A9C25BF357DD4
RUN curl -Lo /usr/local/bin/gosu "https://github.com/tianon/gosu/releases/download/1.2/gosu-$(dpkg --print-architecture)" \
	&& curl -Lo /usr/local/bin/gosu.asc "https://github.com/tianon/gosu/releases/download/1.2/gosu-$(dpkg --print-architecture).asc" \
	&& gpg --verify /usr/local/bin/gosu.asc \
	&& rm /usr/local/bin/gosu.asc \
	&& chmod +x /usr/local/bin/gosu

COPY . /go/src/bitbucket.org/davars/sohop
WORKDIR /go/src/bitbucket.org/davars/sohop
RUN go-wrapper download
RUN go install bitbucket.org/davars/sohop/cmd/sohop

# allow sohop to still bind to ports 80 & 443 even though it's not root
RUN setcap cap_net_bind_service=+ep /go/bin/sohop

EXPOSE 80 443
VOLUME ["/certs"]

ENTRYPOINT ["/usr/local/bin/gosu", "swuser", "/go/bin/sohop"]
