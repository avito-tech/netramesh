FROM golang:1.12

COPY . /go/src/github.com/Lookyan/netramesh
WORKDIR /go/src/github.com/Lookyan/netramesh
RUN go build ./cmd/main.go

ENTRYPOINT ["./main"]
