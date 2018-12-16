package protocol

import "strings"

type Proto string

const (
	HTTPProto Proto = "http"
	TCPProto  Proto = "tcp"
)

func Determine(addr string) Proto {
	port := strings.Split(addr, ":")[1]
	switch port {
	case "80":
		return HTTPProto
	case "8890":
		return HTTPProto
	case "8891":
		return HTTPProto
	case "8080":
		return HTTPProto
	default:
		return TCPProto
	}
}
