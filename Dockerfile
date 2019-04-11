FROM golang:1.11 as builder

COPY . /go/src/github.com/Lookyan/netramesh
WORKDIR /go/src/github.com/Lookyan/netramesh
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o main ./cmd/main.go


FROM alpine
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/github.com/Lookyan/netramesh/main /app/
WORKDIR /app

ENTRYPOINT ["./main"]
