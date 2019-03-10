package protocol

import (
	"strings"

	"github.com/Lookyan/netramesh/internal/config"
)

type Proto string

const (
	HTTPProto Proto = "http"
	TCPProto  Proto = "tcp"
)

func Determine(addr string) Proto {
	httpPorts := config.GetNetraConfig().HTTPProtoPorts
	port := strings.Split(addr, ":")[1]
	if _, ok := httpPorts[port]; ok {
		return HTTPProto
	}
	return TCPProto
}
