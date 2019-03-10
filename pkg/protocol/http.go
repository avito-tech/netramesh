package protocol

import (
	"bufio"
	"container/list"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/patrickmn/go-cache"
	"github.com/uber/jaeger-client-go"

	"github.com/Lookyan/netramesh/internal/config"
	"github.com/Lookyan/netramesh/pkg/log"
)

type HTTPHandler struct {
	tracingContextMapping *cache.Cache
	logger                *log.Logger
}

func NewHTTPHandler(logger *log.Logger, tracingContextMapping *cache.Cache) *HTTPHandler {
	return &HTTPHandler{
		tracingContextMapping: tracingContextMapping,
		logger:                logger,
	}
}

func (h *HTTPHandler) HandleRequest(r io.ReadCloser, w io.WriteCloser, netRequest NetRequest, isInboundConn bool) {
	netHTTPRequest := netRequest.(*NetHTTPRequest)
	netHTTPRequest.isInbound = isInboundConn
	netHTTPRequest.tracingContextMapping = h.tracingContextMapping

	tmpWriter := NewTempWriter()
	defer tmpWriter.Close()
	readerWithFallback := io.TeeReader(r, tmpWriter)
	bufioHTTPReader := bufio.NewReader(readerWithFallback)
	for {
		tmpWriter.Start()
		req, err := http.ReadRequest(bufioHTTPReader)
		if err == io.EOF {
			h.logger.Debug("EOF while parsing request HTTP")
			return
		}
		if err != nil {
			h.logger.Warningf("Error while parsing http request '%s'", err.Error())
			_, err = io.Copy(w, tmpWriter)
			if err != nil {
				h.logger.Warning(err.Error())
			}
			tmpWriter.Stop()
			_, err = io.Copy(w, bufioHTTPReader)
			if err != nil {
				h.logger.Warning(err.Error())
			}
			return
		}
		// avoid ws connections and other upgrade protos
		if strings.ToLower(req.Header.Get("Connection")) == "upgrade" {
			_, err = io.Copy(w, tmpWriter)
			if err != nil {
				h.logger.Warning(err.Error())
			}
			tmpWriter.Stop()
			_, err = io.Copy(w, bufioHTTPReader)
			if err != nil {
				h.logger.Warning(err.Error())
			}
			return
		}

		tmpWriter.Stop()

		// TODO: expose x-request-id key to sidecar httpConfig
		if req.Header.Get("x-request-id") == "" {
			req.Header["X-Request-Id"] = []string{uuid.New().String()}
		}

		if !isInboundConn {
			// we need to generate context header and propagate it
			tracingInfoByRequestID, ok := h.tracingContextMapping.Get(req.Header.Get("x-request-id"))
			if ok {
				h.logger.Debugf("Found request-id matching: %#v", tracingInfoByRequestID)
				tracingContext := tracingInfoByRequestID.(jaeger.SpanContext)
				req.Header[jaeger.TraceContextHeaderName] = []string{tracingContext.String()}
				h.logger.Debugf("Outbound span: %s", tracingContext.String())
			}
		}

		netHTTPRequest.SetHTTPRequest(req)
		netHTTPRequest.StartRequest()

		// write the same request to writer
		err = req.Write(w)
		if err != nil {
			h.logger.Errorf("Error while writing request to w: %s", err.Error())
		}
	}
}

func (h *HTTPHandler) HandleResponse(r io.ReadCloser, w io.WriteCloser, netRequest NetRequest, isInboundConn bool) {
	netHTTPRequest := netRequest.(*NetHTTPRequest)
	netHTTPRequest.isInbound = isInboundConn
	netHTTPRequest.tracingContextMapping = h.tracingContextMapping

	tmpWriter := NewTempWriter()
	defer tmpWriter.Close()
	readerWithFallback := io.TeeReader(r, tmpWriter)
	bufioHTTPReader := bufio.NewReader(readerWithFallback)
	for {
		tmpWriter.Start()
		resp, err := http.ReadResponse(bufioHTTPReader, nil)
		if err == io.EOF {
			h.logger.Debug("EOF while parsing response HTTP")
			netHTTPRequest.StopRequest()
			return
		}
		if err != nil {
			h.logger.Warningf("Error while parsing http response: %s", err.Error())
			_, err = io.Copy(w, tmpWriter)
			if err != nil {
				h.logger.Warning(err.Error())
			}
			tmpWriter.Stop()
			_, err = io.Copy(w, bufioHTTPReader)
			if err != nil {
				h.logger.Warning(err.Error())
			}
			netHTTPRequest.StopRequest()
			return
		}

		// avoid ws connections and other upgrade protos
		if strings.ToLower(resp.Header.Get("Connection")) == "upgrade" {
			_, err = io.Copy(w, tmpWriter)
			if err != nil {
				h.logger.Warning(err.Error())
			}
			tmpWriter.Stop()
			_, err = io.Copy(w, bufioHTTPReader)
			if err != nil {
				h.logger.Warning(err.Error())
			}
			return
		}

		tmpWriter.Stop()

		// write the same response to w
		err = resp.Write(w)
		if err != nil {
			h.logger.Errorf("Error while writing response to w: %s", err.Error())
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
	tracingContextMapping *cache.Cache
	logger                *log.Logger
}

func NewNetHTTPRequest(logger *log.Logger) *NetHTTPRequest {
	return &NetHTTPRequest{
		httpRequests:  NewQueue(),
		httpResponses: NewQueue(),
		spans:         NewQueue(),
		logger:        logger,
	}
}

func (nr *NetHTTPRequest) StartRequest() {
	request := nr.httpRequests.Peek()
	if request == nil {
		return
	}
	httpRequest := request.(*http.Request)
	carrier := opentracing.HTTPHeadersCarrier(httpRequest.Header)
	nr.logger.Debugf("Extraction header value: %s", httpRequest.Header.Get(jaeger.TraceContextHeaderName))
	wireContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)

	operation := httpRequest.URL.Path
	if !nr.isInbound {
		operation = httpRequest.Host + httpRequest.URL.Path
	}
	var span opentracing.Span
	if err != nil {
		nr.logger.Infof("Carrier extract error: %s", err.Error())
		span = opentracing.StartSpan(
			operation,
		)
		if nr.isInbound {
			context := span.Context().(jaeger.SpanContext)
			nr.tracingContextMapping.SetDefault(
				httpRequest.Header.Get("x-request-id"),
				context,
			)
			httpConfig := config.GetHTTPConfig()
			if len(httpConfig.HeadersMap) > 0 {
				// prefer httpConfig iteration, headers are already parsed into a map
				for headerName, tagName := range httpConfig.HeadersMap {
					if val := httpRequest.Header.Get(headerName); val != "" {
						span.SetTag(tagName, val)
					}
				}
			}
			if len(httpConfig.CookiesMap) > 0 {
				// prefer cookies list iteration (there is no pre-parsed cookies list)
				for _, cookie := range httpRequest.Cookies() {
					if tagName, ok := httpConfig.CookiesMap[cookie.Name]; ok {
						span.SetTag(tagName, cookie.Value)
					}
				}
			}
		} else {
			span.Tracer().Inject(
				span.Context(),
				opentracing.HTTPHeaders,
				opentracing.HTTPHeadersCarrier(httpRequest.Header),
			)
		}
	} else {
		nr.logger.Debugf("Wirecontext: %#v", wireContext)
		if nr.isInbound {
			context := wireContext.(jaeger.SpanContext)
			nr.tracingContextMapping.SetDefault(
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
		span := nr.spans.Pop()
		if span != nil {
			requestSpan := span.(opentracing.Span)
			nr.fillSpan(requestSpan, httpRequest, httpResponse)
			requestSpan.Finish()
		}
	}

	if request != nil && response == nil {
		httpRequest := request.(*http.Request)
		span := nr.spans.Pop()
		if span != nil {
			requestSpan := span.(opentracing.Span)
			nr.fillSpan(requestSpan, httpRequest, nil)
			requestSpan.SetTag("error", true)
			requestSpan.SetTag("timeout", true)
			requestSpan.Finish()
		}
	}
}

func (nr *NetHTTPRequest) fillSpan(span opentracing.Span, req *http.Request, resp *http.Response) {
	if nr.isInbound {
		span.SetTag("span.kind", "server")
	} else {
		span.SetTag("span.kind", "client")
	}
	if req != nil {
		span.SetTag("http.host", req.Host)
		span.SetTag("http.path", req.URL.String())
		span.SetTag("http.request_size", req.ContentLength)
		span.SetTag("http.method", req.Method)
		if userAgent := req.Header.Get("User-Agent"); userAgent != "" {
			span.SetTag("http.user_agent", userAgent)
		}
		if requestID := req.Header.Get("X-Request-ID"); requestID != "" {
			span.SetTag("http.request_id", requestID)
		}
	}
	if resp != nil {
		span.SetTag("http.response_size", resp.ContentLength)
		span.SetTag("http.status_code", resp.StatusCode)
		if resp.StatusCode >= 500 {
			span.SetTag("error", "true")
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
