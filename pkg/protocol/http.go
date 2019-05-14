package protocol

import (
	"bufio"
	"bytes"
	"container/list"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/patrickmn/go-cache"
	"github.com/uber/jaeger-client-go"

	"github.com/Lookyan/netramesh/internal/config"
	"github.com/Lookyan/netramesh/pkg/fhttp"
	"github.com/Lookyan/netramesh/pkg/log"
)

var dumbReader = bytes.NewReader([]byte{})
var readerPool = sync.Pool{
	New: func() interface{} {
		return bufio.NewReaderSize(dumbReader, 0xfff)
	},
}

var dumbWriter = bytes.NewBuffer([]byte{})
var writerPool = sync.Pool{
	New: func() interface{} {
		return bufio.NewWriterSize(dumbWriter, 0xfff)
	},
}

var queuePool = sync.Pool{
	New: func() interface{} {
		return NewQueue()
	},
}

var emptyBytes = make([]byte, 0)

// HTTPHandler process HTTP protocol
type HTTPHandler struct {
	tracingContextMapping     *cache.Cache
	routingInfoContextMapping *cache.Cache
	logger                    *log.Logger
}

// NewHTTPHandler returns HTTP handler
func NewHTTPHandler(
	logger *log.Logger,
	tracingContextMapping *cache.Cache,
	routingInfoContextMapping *cache.Cache) *HTTPHandler {
	return &HTTPHandler{
		tracingContextMapping:     tracingContextMapping,
		routingInfoContextMapping: routingInfoContextMapping,
		logger:                    logger,
	}
}

// HandleRequest handles HTTP request
func (h *HTTPHandler) HandleRequest(
	r *net.TCPConn,
	w *net.TCPConn,
	connCh chan *net.TCPConn,
	addrCh chan string,
	netRequest NetRequest,
	isInboundConn bool,
	originalDst string) *net.TCPConn {
	netHTTPRequest := netRequest.(*NetHTTPRequest)
	tmpWriter := NewTempWriter()
	defer tmpWriter.Close()
	readerWithFallback := io.TeeReader(r, tmpWriter)
	bufioHTTPReader := readerPool.Get().(*bufio.Reader)
	bufioHTTPReader.Reset(readerWithFallback)
	defer readerPool.Put(bufioHTTPReader)
	for {
		tmpWriter.Start()

		req := fhttp.RequestsPool.Get().(*fhttp.Request)
		defer func() {
			req.Reset()
			fhttp.RequestsPool.Put(req)
		}()
		err := fhttp.ParseRequestHeaders(req, bufioHTTPReader)
		if err == io.EOF {
			h.logger.Debug("EOF while parsing request HTTP")
			return w
		}
		if w == nil {
			// here we can override destination (DNS allowed)
			if config.GetHTTPConfig().RoutingEnabled && req != nil {
				if v := req.Header.Peek(config.GetHTTPConfig().RoutingHeaderName); string(v) != "" {
					addr, err := getRoutingDestination(string(v), string(req.Host()), originalDst)
					if err != nil {
						log.Warning(err.Error())
						addrCh <- originalDst
					} else {
						addrCh <- addr
					}
				} else {
					addrCh <- originalDst
				}
			} else {
				addrCh <- originalDst
			}

			w = <-connCh
			if w == nil {
				return w
			}

			if isInboundConn {
				netHTTPRequest.remoteAddr = r.RemoteAddr().String()
			} else {
				netHTTPRequest.remoteAddr = w.RemoteAddr().String()
			}
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
			return w
		}
		// avoid ws connections and other upgrade protos
		if bytes.Equal(bytes.ToLower(req.Header.Peek("Connection")), []byte("upgrade")) {
			_, err = io.Copy(w, tmpWriter)
			if err != nil {
				h.logger.Warning(err.Error())
			}
			tmpWriter.Stop()
			_, err = io.Copy(w, bufioHTTPReader)
			if err != nil {
				h.logger.Warning(err.Error())
			}
			return w
		}

		tmpWriter.Stop()

		if bytes.Equal(req.Header.Peek(config.GetHTTPConfig().RequestIdHeaderName), emptyBytes) {
			req.Header.Set(config.GetHTTPConfig().RequestIdHeaderName, uuid.New().String())
		}

		if !isInboundConn {
			// we need to generate context header and propagate it
			tracingInfoByRequestID, ok := h.tracingContextMapping.Get(
				string(req.Header.Peek(config.GetHTTPConfig().RequestIdHeaderName)),
			)
			if ok {
				//h.logger.Debugf("Found request-id matching: %#v", tracingInfoByRequestID)
				tracingContext := tracingInfoByRequestID.(jaeger.SpanContext)
				req.Header.Set(jaeger.TraceContextHeaderName, tracingContext.String())
				//h.logger.Debugf("Outbound span: %s", tracingContext.String())
			}
			if v := req.Header.Peek(config.GetHTTPConfig().XSourceHeaderName); bytes.Equal(v, emptyBytes) {
				req.Header.Set(config.GetHTTPConfig().XSourceHeaderName, config.GetHTTPConfig().XSourceValue)
			}
		}

		netHTTPRequest.SetHTTPRequest(req)
		netHTTPRequest.StartRequest()

		bufioWriter := writerPool.Get().(*bufio.Writer)
		bufioWriter.Reset(w)
		// write the same request to writer
		_, err = fhttp.WriteRequestHeaders(req, bufioWriter)
		if err != nil && err != io.ErrUnexpectedEOF {
			h.logger.Errorf("Error while writing request to w: %s", err.Error())
		}
		_, err = fhttp.WriteRequestHeaders(req, bufioWriter)
		if err != nil && err != io.ErrUnexpectedEOF {
			h.logger.Errorf("Error while writing request to w: %s", err.Error())
		}
		err = fhttp.ParseAndProxyRequestBody(req, bufioHTTPReader, bufioWriter)
		bufioWriter.Flush()
		writerPool.Put(bufioWriter)
		if err != nil && err != io.ErrUnexpectedEOF {
			h.logger.Errorf("Error while writing request to w: %s", err.Error())
		}
	}

	return w
}

func (h *HTTPHandler) HandleResponse(r *net.TCPConn, w *net.TCPConn, netRequest NetRequest, isInboundConn bool) {
	netHTTPRequest := netRequest.(*NetHTTPRequest)
	tmpWriter := NewTempWriter()
	defer tmpWriter.Close()
	readerWithFallback := io.TeeReader(r, tmpWriter)
	bufioHTTPReader := readerPool.Get().(*bufio.Reader)
	bufioHTTPReader.Reset(readerWithFallback)
	defer readerPool.Put(bufioHTTPReader)
	defer netHTTPRequest.CleanUp()
	for {
		tmpWriter.Start()
		resp := fhttp.ResponsePool.Get().(*fhttp.Response)
		err := fhttp.ParseResponseHeaders(resp, bufio.NewReader(bufioHTTPReader))
		if err == io.EOF {
			h.logger.Debug("EOF while parsing response HTTP")
			netHTTPRequest.StopRequest()
			resp.Reset()
			fhttp.ResponsePool.Put(resp)
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
			resp.Reset()
			fhttp.ResponsePool.Put(resp)
			return
		}

		// avoid ws connections and other upgrade protos
		if bytes.Equal(bytes.ToLower(resp.Header.Peek("Connection")), []byte("upgrade")) {
			_, err = io.Copy(w, tmpWriter)
			if err != nil {
				h.logger.Warning(err.Error())
			}
			tmpWriter.Stop()
			_, err = io.Copy(w, bufioHTTPReader)
			if err != nil {
				h.logger.Warning(err.Error())
			}
			resp.Reset()
			fhttp.ResponsePool.Put(resp)
			return
		}

		tmpWriter.Stop()

		// if method == HEAD and content-length != 0, it will hang on read with LimitReader, handle this:
		//rq := netHTTPRequest.httpRequests.Peek()
		//if rq != nil && rq.(*fhttp.Request).Method == fhttp.MethodHead {
		//	err = resp.Write(w)
		//} else {
		bufioWriter := writerPool.Get().(*bufio.Writer)
		bufioWriter.Reset(w)
		// write the same response to w
		_, err = fhttp.WriteResponseHeaders(resp, bufioWriter)
		if err != nil {
			h.logger.Errorf("Error while writing response to w: %s", err.Error())
		}
		err = fhttp.ParseAndProxyResponseBody(resp, bufioHTTPReader, bufioWriter)
		if err != nil {
			h.logger.Errorf("Error while writing response to w: %s", err.Error())
		}

		bufioWriter.Flush()
		writerPool.Put(bufioWriter)
		//}

		netHTTPRequest.SetHTTPResponse(resp)
		netHTTPRequest.StopRequest()
		resp.Reset()
		fhttp.ResponsePool.Put(resp)
	}
}

type NetHTTPRequest struct {
	httpRequests          *Queue
	httpResponses         *Queue
	spans                 *Queue
	isInbound             bool
	tracingContextMapping *cache.Cache
	logger                *log.Logger
	remoteAddr            string
}

func NewNetHTTPRequest(logger *log.Logger, isInbound bool, tracingContextMapping *cache.Cache) *NetHTTPRequest {
	return &NetHTTPRequest{
		httpRequests:          queuePool.Get().(*Queue),
		httpResponses:         queuePool.Get().(*Queue),
		spans:                 queuePool.Get().(*Queue),
		logger:                logger,
		isInbound:             isInbound,
		tracingContextMapping: tracingContextMapping,
	}
}

func (nr *NetHTTPRequest) StartRequest() {
	request := nr.httpRequests.Peek()
	if request == nil {
		return
	}
	httpRequest := request.(*fhttp.Request)
	var ctx jaeger.SpanContext

	carrier := opentracing.HTTPHeadersCarrier(httpRequest.Header)
	wireContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)

	operation := httpRequest.URL.Path
	if !nr.isInbound {
		operation = httpRequest.Host + httpRequest.URL.Path
	}
	httpConfig := config.GetHTTPConfig()
	var span opentracing.Span
	if err != nil {
		nr.logger.Infof("Carrier extract error: %s", err.Error())
		span = opentracing.StartSpan(
			operation,
		)

		if nr.isInbound {
			context := span.Context().(jaeger.SpanContext)
			nr.tracingContextMapping.SetDefault(
				httpRequest.Header.Get(httpConfig.RequestIdHeaderName),
				context,
			)

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
		if nr.isInbound {
			context := wireContext.(jaeger.SpanContext)
			nr.tracingContextMapping.SetDefault(
				httpRequest.Header.Get(httpConfig.RequestIdHeaderName),
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
		httpRequest := request.(*fhttp.Request)
		httpResponse := response.(*fhttp.Response)
		span := nr.spans.Pop()
		if span != nil {
			requestSpan := span.(opentracing.Span)
			nr.fillSpan(requestSpan, httpRequest, httpResponse)
			requestSpan.Finish()
		}
	}

	if request != nil && response == nil {
		httpRequest := request.(*fhttp.Request)
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

func (nr *NetHTTPRequest) CleanUp() {
	queuePool.Put(nr.httpRequests)
	queuePool.Put(nr.httpResponses)
	queuePool.Put(nr.spans)
}

func (nr *NetHTTPRequest) fillSpan(
	span opentracing.Span,
	req *fhttp.Request,
	resp *fhttp.Response) {
	if nr.isInbound {
		span.SetTag("span.kind", "server")
	} else {
		span.SetTag("span.kind", "client")
	}
	span.SetTag("remote_addr", nr.remoteAddr)
	if req != nil {
		span.SetTag("http.host", req.Host)
		span.SetTag("http.path", req.URL.String())
		span.SetTag("http.request_size", req.ContentLength)
		span.SetTag("http.method", req.Method)
		if userAgent := req.Header.Get("User-Agent"); userAgent != "" {
			span.SetTag("http.user_agent", userAgent)
		}
		if requestID := req.Header.Get(config.GetHTTPConfig().RequestIdHeaderName); requestID != "" {
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

func (nr *NetHTTPRequest) SetHTTPRequest(r *fhttp.Request) {
	nr.httpRequests.Push(r)
}

func (nr *NetHTTPRequest) SetHTTPResponse(r *fhttp.Response) {
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

// Clear clears queue
func (q *Queue) Clear() {
	for el := q.Pop(); el != nil; {
	}
}

func getRoutingDestination(routingValue string, host string, originalDst string) (string, error) {
	pairs := strings.Split(routingValue, ";")
	for _, p := range pairs {
		keyval := strings.Split(p, "=")
		if len(keyval) < 2 {
			return "", fmt.Errorf("malformed routing header: '%s'", routingValue)
		}
		if host == keyval[0] {
			if !strings.Contains(keyval[1], ":") {
				keyval[1] += ":80"
			}
			return keyval[1], nil
		}
	}
	return originalDst, nil
}

func ExtractTracingFromHeader(req *fhttp.Request) jaeger.SpanContext {
	var span jaeger.SpanContext
	req.Header.Peek("")
}
