package protocol

import (
	"io"
	"net"

	"github.com/Lookyan/netramesh/pkg/log"
)

type TCPHandler struct {
	logger *log.Logger
}

func NewTCPHandler(logger *log.Logger) *TCPHandler {
	return &TCPHandler{
		logger: logger,
	}
}

func (h *TCPHandler) HandleRequest(r *net.TCPConn, connCh chan *net.TCPConn, addrCh chan string, netRequest NetRequest, isInboundConn bool, originalDst string) {
	addrCh <- originalDst
	w := <-connCh

	buf := bufferPool.Get().([]byte)
	written, err := io.CopyBuffer(w, r, buf)
	bufferPool.Put(buf)
	h.logger.Debugf("Written: %d", written)
	if err != nil {
		h.logger.Debugf("Err CopyBuffer: %s", err.Error())
	}
}

func (h *TCPHandler) HandleResponse(r *net.TCPConn, w *net.TCPConn, netRequest NetRequest, isInboundConn bool) {
	buf := bufferPool.Get().([]byte)
	written, err := io.CopyBuffer(w, r, buf)
	bufferPool.Put(buf)
	h.logger.Debugf("Written: %d", written)
	if err != nil {
		h.logger.Debugf("Err CopyBuffer: %s", err.Error())
	}
}

type NetTCPRequest struct {
}

func NewNetTCPRequest(logger *log.Logger) *NetTCPRequest {
	return &NetTCPRequest{}
}

func (r *NetTCPRequest) StartRequest() {

}

func (r *NetTCPRequest) StopRequest() {

}
