package protocol

import (
	"io"
	"log"
)

type TCPHandler struct {
}

func NewTCPHandler() *TCPHandler {
	return &TCPHandler{}
}

func (h *TCPHandler) HandleRequest(r io.ReadCloser, w io.WriteCloser, netRequest NetRequest, isInBoundConn bool) {
	buf := bufferPool.Get().([]byte)
	written, err := io.CopyBuffer(w, r, buf)
	bufferPool.Put(buf)
	log.Printf("Written: %d", written)
	if err != nil {
		log.Printf("Err CopyBuffer: %s", err.Error())
	}
}

func (h *TCPHandler) HandleResponse(r io.ReadCloser, w io.WriteCloser, netRequest NetRequest, isInBoundConn bool) {
	buf := bufferPool.Get().([]byte)
	written, err := io.CopyBuffer(w, r, buf)
	bufferPool.Put(buf)
	log.Printf("Written: %d", written)
	if err != nil {
		log.Printf("Err CopyBuffer: %s", err.Error())
	}
}

type NetTCPRequest struct {
}

func NewNetTCPRequest() *NetTCPRequest {
	return &NetTCPRequest{}
}

func (r *NetTCPRequest) StartRequest() {

}

func (r *NetTCPRequest) StopRequest() {

}
