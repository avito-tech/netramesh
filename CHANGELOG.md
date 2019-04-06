# 0.4
- Added remote_addr tag to Jaeger
- Added X-Source header to propagate origin service name which can be customized through `NETRA_HTTP_X_SOURCE_HEADER_NAME` and `NETRA_HTTP_X_SOURCE_VALUE` env variable

# 0.3.1
- HTTP parsing moved to vendored stdlib
- HTTP WriteBody optimized (CopyBuffer used instead of Copy)

# 0.3
- X-Request-Id exposed to ENV VAR

# 0.2.2
- Fixed HTTP HEAD check

# 0.2.1
- Fixed bug with HTTP HEAD responses with non zero Content-length header
- HTTP stdlib vendored

# 0.2
- Added probabilistic routing mechanism

# 0.1.3
- Avoid address allocations in connection handling

# 0.1.2
- Performance optimizations

# 0.1.1
- Improve performance of http handler. Avoid additional allocations.

# 0.1
- Logger, configuration and specific port traffic forwarding

# 0.0.7
- Added timeout tag to spans for client timeouts

# 0.0.6
- Added HTTP_HEADER_TAG_MAP and HTTP_COOKIE_TAG_MAP Env config vars for HTTP headers to tracing span tag conversion

# 0.0.5
- HTTP upgrade type fallback to TCP added

# 0.0.4
- Added TCP fallback when we can't handle http protocol parsing.

# 0.0.3
- Added X-Request-id header detection and tracing span context propagation.

# 0.0.2
- Fixed read deadlock when client doesn't close connection by itself. Now it supports HTTP/1.0 proto.

# 0.0.1
- Base functionality for tcp traffic proxy and HTTP/1.1 parsing. Jaeger tracing spans support. 
