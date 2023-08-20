FROM golang:1.20 as builder

ENV GO111MODULE=on \
	GOOS=linux \
	GOARCH=amd64 \
	CGO_ENABLED=0

RUN mkdir /build
WORKDIR /build
ADD . .

RUN cd example && go build --ldflags '-extldflags "-static"'

FROM scratch

USER 1000:1000
WORKDIR /bin/

COPY --from=builder --chown=1000:1000 /etc/ssl/certs /etc/ssl/certs
COPY --from=builder --chown=1000:1000 /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder --chown=1000:1000 /build/example/example ./example

ENTRYPOINT ["/bin/example"]
