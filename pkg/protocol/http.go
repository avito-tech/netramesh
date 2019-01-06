package protocol

import (
	"bufio"
	"container/list"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
)

type HTTPHandler struct {
	tracingContextMapping *sync.Map
}

func NewHTTPHandler(tracingContextMapping *sync.Map) *HTTPHandler {
	return &HTTPHandler{
		tracingContextMapping: tracingContextMapping,
	}
}

func (h *HTTPHandler) HandleRequest(r io.ReadCloser, w io.WriteCloser, netRequest NetRequest, isInboundConn bool) {
	netHTTPRequest := netRequest.(*NetHTTPRequest)
	netHTTPRequest.isInbound = isInboundConn
	netHTTPRequest.tracingContextMapping = h.tracingContextMapping
	bufioHTTPReader := bufio.NewReader(r)
	for {
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

		// TODO: expose x-request-id key to sidecar config
		if req.Header.Get("x-request-id") == "" {
			req.Header["X-Request-Id"] = []string{uuid.New().String()}
		}

		if !isInboundConn {
			// we need to generate context header and propagate it
			tracingInfoByRequestID, ok := h.tracingContextMapping.Load(req.Header.Get("x-request-id"))
			if ok {
				log.Printf("Found request-id matching: %#v", tracingInfoByRequestID)
				tracingContext := tracingInfoByRequestID.(jaeger.SpanContext)
				req.Header[jaeger.TraceContextHeaderName] = []string{tracingContext.String()}
				log.Printf("Outbound span: %s", tracingContext.String())
			}
		}

		netHTTPRequest.SetHTTPRequest(req)
		netHTTPRequest.StartRequest()

		// write the same request to writer
		err = req.Write(w)
		if err != nil {
			log.Printf("Error while writing request to w: %s", err.Error())
		}
	}
}

func (h *HTTPHandler) HandleResponse(r io.ReadCloser, w io.WriteCloser, netRequest NetRequest, isInboundConn bool) {
	netHTTPRequest := netRequest.(*NetHTTPRequest)
	netHTTPRequest.isInbound = isInboundConn
	netHTTPRequest.tracingContextMapping = h.tracingContextMapping
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
	httpRequests          *Queue
	httpResponses         *Queue
	spans                 *Queue
	isInbound             bool
	tracingContextMapping *sync.Map
}

func NewNetHTTPRequest() *NetHTTPRequest {
	return &NetHTTPRequest{
		httpRequests:  NewQueue(),
		httpResponses: NewQueue(),
		spans:         NewQueue(),
	}
}

func (nr *NetHTTPRequest) StartRequest() {
	request := nr.httpRequests.Peek()
	if request == nil {
		return
	}
	httpRequest := request.(*http.Request)
	carrier := opentracing.HTTPHeadersCarrier(httpRequest.Header)
	log.Printf("Extraction header value: %s", httpRequest.Header.Get(jaeger.TraceContextHeaderName))
	wireContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)

	operation := httpRequest.URL.Path
	if !nr.isInbound {
		operation = httpRequest.Host + httpRequest.URL.Path
	}
	var span opentracing.Span
	if err != nil {
		log.Printf("Carrier extract error: %s", err.Error())
		span = opentracing.StartSpan(
			operation,
		)
		if nr.isInbound {
			context := span.Context().(jaeger.SpanContext)
			nr.tracingContextMapping.Store(
				httpRequest.Header.Get("x-request-id"),
				context,
			)
		} else {
			span.Tracer().Inject(
				span.Context(),
				opentracing.HTTPHeaders,
				opentracing.HTTPHeadersCarrier(httpRequest.Header),
			)
		}
	} else {
		log.Printf("Wirecontext: %#v", wireContext)
		if nr.isInbound {
			context := wireContext.(jaeger.SpanContext)
			nr.tracingContextMapping.Store(
				httpRequest.Header.Get("x-request-id"),
				context,
			)
		}
		span = opentracing.StartSpan(
			operation,
			opentracing.ChildOf(wireContext),
		)
	}

	nr.spans.Push(span)
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

			requestSpan.SetTag("http.host", httpRequest.Host)
			requestSpan.SetTag("http.path", httpRequest.URL.String())
			requestSpan.SetTag("http.request_size", httpRequest.ContentLength)
			requestSpan.SetTag("http.response_size", httpResponse.ContentLength)
			requestSpan.SetTag("http.method", httpRequest.Method)
			requestSpan.SetTag("http.status_code", httpResponse.StatusCode)

			if nr.isInbound {
				requestSpan.SetTag("span.kind", "server")
			} else {
				requestSpan.SetTag("span.kind", "client")
			}
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
func NewQueue() *Queue {
	return &Queue{
		elements: list.New(),
	}
}

// Queue implements queue data structure
type Queue struct {
	mu       sync.Mutex
	elements *list.List
}

// Push pushes element to the end of queue
func (q *Queue) Push(value interface{}) {
	q.mu.Lock()
	q.elements.PushBack(value)
	q.mu.Unlock()
}

// Pop pops first element out of queue
func (q *Queue) Pop() interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()
	el := q.elements.Front()
	if el == nil {
		return nil
	}
	return q.elements.Remove(el)
}

// Peek returns first element in the queue without removing it
func (q *Queue) Peek() interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()
	if el := q.elements.Front(); el != nil {
		return el.Value
	} else {
		return nil
	}
}
