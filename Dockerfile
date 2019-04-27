FROM golang:1.12 AS builder

WORKDIR /src

ADD . .

ENV GOOS        linux
ENV GOARCH      amd64
ENV CGO_ENABLED 0

RUN go build  -o /go/bin/netramesh \
              -mod vendor \
              -a -installsuffix cgo \
              -ldflags '-extldflags "-static"' \
              ./cmd/main.go


FROM alpine:latest AS service

LABEL maintainers="Alexander Lukyanchenko <digwnews@gmail.com>, \
Mikhail Leonov <lm@kodix.ru>, \
Kamil Samigullin <kamil@samigullin.info>"

RUN adduser -D -H -u 1000 service

USER service

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/netramesh /usr/local/bin/

ENTRYPOINT [ "netramesh" ]
CMD        [ "-h" ]
