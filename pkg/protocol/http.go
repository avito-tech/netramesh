package protocol

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/opentracing/opentracing-go"
)

type HTTPHandler struct {

}

func NewHTTPHandler() *HTTPHandler {
	return &HTTPHandler{
	}
}

func (h *HTTPHandler) HandleRequest(pr *io.PipeReader, pw *io.PipeWriter, netRequest NetRequest) {
	defer pr.Close()
	defer pw.Close()
	netHTTPRequest := netRequest.(*NetHTTPRequest)
	for {
		netHTTPRequest.StartRequest()
		bufioHTTPReader := bufio.NewReader(pr)

		var r *http.Request
		r, err := http.ReadRequest(bufioHTTPReader)
		log.Printf("%#v \n", r)
		if err == io.EOF {
			log.Print("EOF while parsing request HTTP")
			return
		}
		if err != nil {
			log.Printf("Error while parsing http '%s'", err.Error())
			io.Copy(ioutil.Discard, bufioHTTPReader)
			return
		}
		if r.Body != http.NoBody {
			n, _ := io.Copy(ioutil.Discard, r.Body)
			netHTTPRequest.requestSize = n
			r.Body.Close()
		}
		netHTTPRequest.SetHTTPRequest(r)
	}
}

func (h *HTTPHandler) HandleResponse(pr *io.PipeReader, pw *io.PipeWriter, netRequest NetRequest) {
	defer pr.Close()
	defer pw.Close()
	netHTTPRequest := netRequest.(*NetHTTPRequest)
	for {
		bufioHTTPReader := bufio.NewReader(pr)
		r, err := http.ReadResponse(bufioHTTPReader, nil)
		if err == io.EOF {
			log.Print("EOF while parsing request HTTP")
			return
		}
		if err != nil {
			log.Printf("Error while parsing response: %s", err.Error())
			io.Copy(ioutil.Discard, bufioHTTPReader)
			return
		}
		log.Printf("Response: %#v", r)
		if r.Body != http.NoBody {
			n, _ := io.Copy(ioutil.Discard, r.Body)
			netHTTPRequest.responseSize = n
			r.Body.Close()
		}
		netHTTPRequest.SetHTTPResponse(r)
		netHTTPRequest.StopRequest()
	}
}

type NetHTTPRequest struct {
	requestTime  time.Time
	lastDuration time.Duration
	httpRequest  *http.Request
	httpResponse *http.Response
	requestSize  int64
	responseSize int64
	span         opentracing.Span
}

func NewNetHTTPRequest() *NetHTTPRequest {
	return &NetHTTPRequest{
	}
}

func (nr *NetHTTPRequest) StartRequest() {
	nr.requestTime = time.Now()
	nr.span = opentracing.StartSpan(
		"netra-span", // sets temporarily
	)
}

func (nr *NetHTTPRequest) StopRequest() {
	nr.lastDuration = time.Since(nr.requestTime)
	if nr.httpRequest != nil {
		log.Printf("Method: %s Host: %s",
			nr.httpRequest.Method,
			nr.httpRequest.Host,
		)
		log.Printf("HTTP request latency: %d", nr.lastDuration)
		if nr.span != nil {
			nr.span.SetOperationName(nr.httpRequest.Host + nr.httpRequest.URL.Path)
			nr.span.SetTag("http.host", nr.httpRequest.Host)
			nr.span.SetTag("http.path", nr.httpRequest.URL.String())
			nr.span.SetTag("http.request_size", nr.requestSize)
			nr.span.SetTag("http.response_size", nr.responseSize)
			nr.span.SetTag("http.method", nr.httpRequest.Method)
			nr.span.SetTag("http.status_code", nr.httpResponse.StatusCode)
			nr.span.SetTag("span.kind", "client")
			if userAgent := nr.httpRequest.Header.Get("User-Agent"); userAgent != "" {
				nr.span.SetTag("http.user_agent", userAgent)
			}
			if requestID := nr.httpRequest.Header.Get("X-Request-ID"); requestID != "" {
				nr.span.SetTag("http.request_id", requestID)
			}
			nr.span.Finish()
		}
	}
}

func (nr *NetHTTPRequest) SetHTTPRequest(r *http.Request) {
	nr.httpRequest = r
}

func (nr *NetHTTPRequest) SetHTTPResponse(r *http.Response) {
	nr.httpResponse = r
}
