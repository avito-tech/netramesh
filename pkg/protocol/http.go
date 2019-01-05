package protocol

import (
	"bufio"
	"container/list"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
)

type HTTPHandler struct {
	tracingContextMapping sync.Map
}

func NewHTTPHandler(tracingContextMapping sync.Map) *HTTPHandler {
	return &HTTPHandler{
		tracingContextMapping: tracingContextMapping,
	}
}

func (h *HTTPHandler) HandleRequest(r io.ReadCloser, w io.WriteCloser, netRequest NetRequest, isInBoundConn bool) {
	netHTTPRequest := netRequest.(*NetHTTPRequest)
	bufioHTTPReader := bufio.NewReader(r)
	for {
		if !isInBoundConn {
			netHTTPRequest.StartRequest()
		}
		req, err := http.ReadRequest(bufioHTTPReader)
		if err == io.EOF {
			log.Print("EOF while parsing request HTTP")
			return
		}
		if err != nil {
			// TODO: fallback to tcp proxy
			log.Printf("Error while parsing http request '%s'", err.Error())
			io.Copy(w, bufioHTTPReader)
			return
		}

		if isInBoundConn {
			if req.Header.Get("x-request-id") != "" {

			} else {
				req.Header["X-Request-Id"] = []string{"uuid4"}
			}
			h.tracingContextMapping.Store(req.Header.Get("x-request-id"), true)
		} else {

		}

		if req.Header.Get(jaeger.TraceContextHeaderName) == "" {
			req.Header[jaeger.TraceContextHeaderName] = []string{"123:123:123:123"}
		}

		// write the same request to writer
		err = req.Write(w)
		if err != nil {
			log.Printf("Error while writing request to w: %s", err.Error())
		}

		if !isInBoundConn {
			netHTTPRequest.SetHTTPRequest(req)
		}
	}
}

func (h *HTTPHandler) HandleResponse(r io.ReadCloser, w io.WriteCloser, netRequest NetRequest, isInBoundConn bool) {
	netHTTPRequest := netRequest.(*NetHTTPRequest)
	bufioHTTPReader := bufio.NewReader(r)
	for {
		resp, err := http.ReadResponse(bufioHTTPReader, nil)
		if err == io.EOF {
			log.Print("EOF while parsing request HTTP")
			netHTTPRequest.StopRequest()
			return
		}
		if err != nil {
			// TODO: fallback to tcp proxy
			log.Printf("Error while parsing http response: %s", err.Error())
			io.Copy(w, bufioHTTPReader)
			netHTTPRequest.StopRequest()
			return
		}

		// write the same response to w
		err = resp.Write(w)
		if err != nil {
			log.Printf("Error while writing response to w: %s", err.Error())
		}

		netHTTPRequest.SetHTTPResponse(resp)
		netHTTPRequest.StopRequest()
	}
}

type NetHTTPRequest struct {
	httpRequests  *Queue
	httpResponses *Queue
	spans         *Queue
}

func NewNetHTTPRequest() *NetHTTPRequest {
	return &NetHTTPRequest{
		httpRequests:  NewQueue(),
		httpResponses: NewQueue(),
		spans:         NewQueue(),
	}
}

func (nr *NetHTTPRequest) StartRequest() {
	nr.spans.Push(opentracing.StartSpan(
		"netra-span", // sets temporarily
	))
}

func (nr *NetHTTPRequest) StopRequest() {
	request := nr.httpRequests.Pop()
	response := nr.httpResponses.Pop()
	if request != nil && response != nil {
		httpRequest := request.(*http.Request)
		httpResponse := response.(*http.Response)
		log.Printf("Method: %s Host: %s",
			httpRequest.Method,
			httpRequest.Host,
		)
		span := nr.spans.Pop()
		if span != nil {
			requestSpan := span.(opentracing.Span)
			requestSpan.SetOperationName(httpRequest.Host + httpRequest.URL.Path)
			carrier := opentracing.HTTPHeadersCarrier(httpRequest.Header)
			wireContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
			if err != nil {
				log.Printf("Carrier extract error: %s", err.Error())
			} else {
				err = requestSpan.Tracer().Inject(wireContext, opentracing.HTTPHeaders, carrier)
				if err != nil {
					log.Printf("Carrier inject error: %s", err.Error())
				}
			}

			requestSpan.SetTag("http.host", httpRequest.Host)
			requestSpan.SetTag("http.path", httpRequest.URL.String())
			requestSpan.SetTag("http.request_size", httpRequest.ContentLength)
			requestSpan.SetTag("http.response_size", httpResponse.ContentLength)
			requestSpan.SetTag("http.method", httpRequest.Method)
			requestSpan.SetTag("http.status_code", httpResponse.StatusCode)
			requestSpan.SetTag("span.kind", "client")
			if userAgent := httpRequest.Header.Get("User-Agent"); userAgent != "" {
				requestSpan.SetTag("http.user_agent", userAgent)
			}
			if requestID := httpRequest.Header.Get("X-Request-ID"); requestID != "" {
				requestSpan.SetTag("http.request_id", requestID)
			}
			requestSpan.Finish()
		}
	}
}

func (nr *NetHTTPRequest) SetHTTPRequest(r *http.Request) {
	nr.httpRequests.Push(r)
}

func (nr *NetHTTPRequest) SetHTTPResponse(r *http.Response) {
	nr.httpResponses.Push(r)
}

// NewQueue creates new queue
// TODO: make it thread-safe
func NewQueue() *Queue {
	return &Queue{
		elements: list.New(),
	}
}

// Queue implements queue data structure
type Queue struct {
	elements *list.List
}

// Push pushes element to the end of queue
func (q *Queue) Push(value interface{}) {
	q.elements.PushBack(value)
}

// Pop pops first element out of queue
func (q *Queue) Pop() interface{} {
	el := q.elements.Front()
	if el == nil {
		return nil
	}
	return q.elements.Remove(el)
}
