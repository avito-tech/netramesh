# netramesh

![netramesh](media/logo.png)

[![CircleCI](https://circleci.com/gh/avito-tech/netramesh/tree/master.svg?style=svg)](https://circleci.com/gh/avito-tech/netramesh/tree/master)

Ultra light service mesh has main goals:

- high performance
- observability (Jaeger distributed tracing)
- simplicity of operation
- unlimited scalability
- any infrastructure compatibility
- transparency

Service mesh netramesh consists of two main parts:
- Transparent TCP proxy for microservices with original destination retrieval.
- Init container for network rules configuration (iptables based).

## Getting started

Check out [examples](./examples)

## Supported application level protocols
- HTTP/1.1 and lower

Also netra supports any TCP proto traffic (proxies it transparently).


## How it works

![main parts](media/netra_main_parts.png)

To intercept all TCP traffic netra uses [iptables redirect rules](./iptables-rules.sh). After applying them, TCP traffic goes firstly to netra sidecar. Netra sidecar determines original destination using SO_ORIGINAL_DST socket option. After that netra sidecar works in bidirectional stream processing mode and proxies all TCP packets through itself. If app level protocol is HTTP1, netra parses it and sends tracing span.

![traffic interception](media/netra_traffic_intercept.png)

## Injecting

For now netra supports only manual injecting.

## Basic configuration (environment variables)

### Netra init (network interception settings)

Env name| Description
---|---
NETRA_SIDECAR_PORT | netra sidecar listen port redirect to (defaults to 14956)
NETRA_SIDECAR_USER_ID | netra sidecar user id to avoid infinite redirect loops (defaults to 1337)
NETRA_SIDECAR_GROUP_ID | netra sidecar group id to avoid infinite redirect loops (defaults to 1337)
INBOUND_INTERCEPT_PORTS | inbound ports to intercept (defaults to *, all ports)
OUTBOUND_INTERCEPT_PORTS | outbound ports to intercept (defaults to *, all ports)
NETRA_INBOUND_PROBABILITY | inbound probability to route TCP sessions (defaults to 1)
NETRA_OUTBOUND_PROBABILITY | outbound probability to route TCP sessions (defaults to 1)


### Netra sidecar

Switches

Switch name| Description
---|---
--service-name| service name for jaeger distributed trace spans

Env name| Description
---|---
NETRA_LOGGER_LEVEL | logger level (defaults to info), supported values: debug, info, warning, error, fatal
NETRA_PORT | netra sidecar listen port (defaults to 14956)
NETRA_PPROF_PORT | netra sidecar pprof port (defaults to 14957)
NETRA_PROMETHEUS_PORT | netra prometheus port (defaults to 14958)
NETRA_TRACING_CONTEXT_EXPIRATION_MILLISECONDS | tracing context mapping cache expiration in milliseconds (defaults to 5000)
NETRA_TRACING_CONTEXT_CLEANUP_INTERVAL | tracing context cleanup interval in milliseconds (defaults to 1000)
NETRA_STATSD_ENABLED | enabling statsd. Set "true" to enable (defaults to false)
NETRA_STATSD_PREFIX | Statsd prefix for all metrics (defaults to "")
NETRA_STATSD_ADDRESS | Statsd gate (defaults to "")
NETRA_HTTP_PORTS | comma separated ports to determine as HTTP1 protocol (no default)
NETRA_HTTP_REQUEST_ID_HEADER_NAME | header name to match inbound and outbound requests. Applications should propagate it (defaults to X-Request-Id)
HTTP_HEADER_TAG_MAP | comma separated HTTP header to jaeger span tag conversion (example: `x-session:http.session,x-mobile-info:http.x-mobile-info`)
HTTP_COOKIE_TAG_MAP | comma separated HTTP cookie value to span tag conversion (example: `sess:http.cookies.sess`)
NETRA_HTTP_X_SOURCE_HEADER_NAME | source HTTP header name. Automatically added to each outbound request in case this header absent in request (defaults to X-Source)
NETRA_HTTP_X_SOURCE_VALUE | source HTTP header value (defaults to netra)
NETRA_HTTP_ROUTING_ENABLED | set this to value "true" to enable HTTP header routing feature (disabled by default)
NETRA_HTTP_ROUTING_HEADER_NAME | header name for HTTP header routing (defaults to `X-Route`). Value of header should be in the following format: `host1=host2,host3=host4` to route host1 to host2 and host3 to host4.
NETRA_ROUTING_CONTEXT_EXPIRATION_MILLISECONDS | routing context mapping cache expiration in milliseconds (defaults to 5000)
NETRA_ROUTING_CONTEXT_CLEANUP_INTERVAL | routing context cleanup interval in milliseconds (defaults to 1000)
NETRA_HTTP_ROUTING_COOKIE_ENABLED | set this to value "true" to enable routing logic from HTTP Cookie (should be enabled with NETRA_HTTP_ROUTING_ENABLED). Cookie has priority to routing HTTP header (disabled by default)
NETRA_HTTP_ROUTING_COOKIE_NAME | cookie name for routing (defaults to `X-Route`)


Also it supports all env variables [jaeger go library](https://github.com/jaegertracing/jaeger-client-go#environment-variables) provides.

## Comparison with Istio and linkerd2

Why do we need one more service mesh solution? Istio and linkerd2 are perfect service mesh solutions with very powerful set of features. But unfortunately they add significant resource and performance overhead.
Netramesh main goal is providing observability to your distributed system with small overhead (approximately 10-50Mb on each netra sidecar) and 1ms of latency overhead. If you don't need entire set of features Istio and linkerd2 provide, but you need to collect distributed traces and obtain important information about your microservice interaction then netra is a great fit.
