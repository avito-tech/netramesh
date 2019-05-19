FROM golang:alpine AS builder

WORKDIR /go/src/github.com/Lookyan/netramesh

ENV GOOS        linux
ENV GOARCH      amd64
ENV CGO_ENABLED 0

ENV GO111MODULE off

RUN apk add --no-cache ca-certificates \
        dpkg \
        gcc \
        git \
        musl-dev \
    && mkdir -p "$GOPATH/src" "$GOPATH/bin" \
    && chmod -R 777 "$GOPATH" \
    && go get github.com/derekparker/delve/cmd/dlv

ENV GO111MODULE on

ADD . .

RUN go build  -o /go/bin/netramesh \
              -mod vendor \
              -gcflags "all=-N -l" \
              ./cmd/main.go

ENV GO111MODULE off

ENV GOPATH /go
WORKDIR /go/src/github.com/Lookyan/netramesh

RUN chmod -R 777 ./

CMD ["dlv", "--headless", "--listen=:2345", "--api-version=2", "exec", "/go/bin/netramesh", "--", "--service-name", "nginx"]
