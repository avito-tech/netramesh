module github.com/Lookyan/netramesh

require (
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/google/uuid v1.1.0
	github.com/opentracing/opentracing-go v1.0.2
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.8.0 // indirect
	github.com/prometheus/client_golang v0.9.2
	github.com/stretchr/testify v1.3.0 // indirect
	github.com/uber-go/atomic v1.3.2 // indirect
	github.com/uber/jaeger-client-go v2.15.0+incompatible
	github.com/uber/jaeger-lib v1.5.0 // indirect
	go.uber.org/atomic v1.3.2 // indirect
	golang.org/x/net v0.0.0-20181201002055-351d144fa1fc
	golang.org/x/text v0.3.2 // indirect
)

replace golang.org/x/crypto => ./internal/patches/golang_org/x/crypto

replace golang.org/x/net => ./internal/patches/golang_org/x/net

replace golang.org/x/text => ./internal/patches/golang_org/x/text
