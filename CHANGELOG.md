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
