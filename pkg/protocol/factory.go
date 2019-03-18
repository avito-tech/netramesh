package protocol

import (
	"sync"

	"github.com/patrickmn/go-cache"

	"github.com/Lookyan/netramesh/pkg/log"
)

var httpHandler *HTTPHandler
var tcpHandler *TCPHandler
var netTCPRequest *NetTCPRequest
var handlerAssignmentLock = sync.Mutex{}
var requestAssignmentLock = sync.Mutex{}

func GetNetworkHandler(proto Proto, logger *log.Logger, tracingContextMapping *cache.Cache) NetHandler {
	handlerAssignmentLock.Lock()
	if httpHandler == nil {
		httpHandler = NewHTTPHandler(logger, tracingContextMapping)
	}
	if tcpHandler == nil {
		tcpHandler = NewTCPHandler(logger)
	}
	handlerAssignmentLock.Unlock()
	switch proto {
	case HTTPProto:
		return httpHandler
	case TCPProto:
		return tcpHandler
	default:
		return tcpHandler
	}
}

func GetNetRequest(proto Proto, isInbound bool, logger *log.Logger, tracingContextMapping *cache.Cache) NetRequest {
	requestAssignmentLock.Lock()
	if netTCPRequest == nil {
		netTCPRequest = NewNetTCPRequest(logger)
	}
	requestAssignmentLock.Unlock()
	switch proto {
	case HTTPProto:
		return NewNetHTTPRequest(logger, isInbound, tracingContextMapping)
	case TCPProto:
		return netTCPRequest
	default:
		return netTCPRequest
	}
}
