package protocol

import (
	"io"
	"io/ioutil"
)

type TCPHandler struct {
}

func NewTCPHandler() *TCPHandler {
	return &TCPHandler{
	}
}

func (h *TCPHandler) HandleRequest(pr *io.PipeReader, pw *io.PipeWriter, netRequest NetRequest) {
	defer pr.Close()
	defer pw.Close()
	io.Copy(ioutil.Discard, pr)
}

func (h *TCPHandler) HandleResponse(pr *io.PipeReader, pw *io.PipeWriter, netRequest NetRequest) {
	defer pr.Close()
	defer pw.Close()
	io.Copy(ioutil.Discard, pr)
}

type NetTCPRequest struct {
}

func NewNetTCPRequest() *NetTCPRequest {
	return &NetTCPRequest{
	}
}

func (r *NetTCPRequest) StartRequest() {

}

func (r *NetTCPRequest) StopRequest() {

}
